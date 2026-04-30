package mdori

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const usageText = "usage: mdori [--addr host:port] [--no-open] <markdown-file>\n"

type config struct {
	addr        string
	openBrowser bool
	path        string
}

type app struct {
	sourcePath string
	renderer   *renderer
	reloader   *reloader
}

func Main(args []string, stdout, stderr io.Writer) int {
	if err := Run(context.Background(), args, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "mdori: %v\n", err)
		return 1
	}

	return 0
}

func Run(parent context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, err := parseArgs(args, stderr)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	signalCtx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	reloader, err := newReloader(signalCtx, cfg.path)
	if err != nil {
		return fmt.Errorf("watch markdown file: %w", err)
	}
	defer reloader.Close()

	server := &http.Server{
		Handler: (&app{
			sourcePath: cfg.path,
			renderer:   newRenderer(),
			reloader:   reloader,
		}).routes(),
	}

	serverErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	url := "http://" + listener.Addr().String()
	fmt.Fprintf(stdout, "Serving at %s\n", url)

	if cfg.openBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(stderr, "mdori: could not open browser automatically: %v\n", err)
		}
	}

	select {
	case err := <-serverErr:
		return err
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		return <-serverErr
	}
}

func parseArgs(args []string, stderr io.Writer) (config, error) {
	cfg := config{
		addr:        "127.0.0.1:0",
		openBrowser: true,
	}

	fs := flag.NewFlagSet("mdori", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.addr, "addr", cfg.addr, "listen address")
	fs.Usage = func() {
		fmt.Fprint(stderr, usageText)
		fs.PrintDefaults()
	}

	var noOpen bool
	fs.BoolVar(&noOpen, "no-open", false, "do not open the browser automatically")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	cfg.openBrowser = !noOpen

	if fs.NArg() != 1 {
		fs.Usage()
		return config{}, errors.New("expected exactly one markdown file")
	}

	path, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return config{}, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return config{}, fmt.Errorf("stat %q: %w", path, err)
	}

	if info.IsDir() {
		return config{}, fmt.Errorf("%q is a directory", path)
	}

	cfg.path = path
	return cfg, nil
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.serveDocument)
	mux.HandleFunc("/events", a.serveEvents)
	return mux
}

func (a *app) serveDocument(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	source, err := os.ReadFile(a.sourcePath)
	if err != nil {
		http.Error(w, "failed to read markdown file", http.StatusInternalServerError)
		return
	}

	rendered, err := a.renderer.render(source)
	if err != nil {
		http.Error(w, "failed to render markdown", http.StatusInternalServerError)
		return
	}

	page, err := renderPage(pageTitle(a.sourcePath), rendered)
	if err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

func (a *app) serveEvents(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/events" {
		http.NotFound(w, r)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	events := a.reloader.Subscribe()
	defer a.reloader.Unsubscribe(events)

	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-events:
			if !ok {
				return
			}

			_, _ = io.WriteString(w, "event: reload\ndata: reload\n\n")
			flusher.Flush()
		}
	}
}

func pageTitle(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
