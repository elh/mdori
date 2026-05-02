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
	"path"
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
	rootDir    string
	root       *os.Root
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

	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	if !pathWithinRoot(rootDir, cfg.path) {
		return fmt.Errorf("markdown file %q is outside serving root %q", cfg.path, rootDir)
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("open serving root: %w", err)
	}
	defer root.Close()

	server := &http.Server{
		Handler: (&app{
			sourcePath: cfg.path,
			rootDir:    rootDir,
			root:       root,
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
		if err := openBrowser(url + rootRelativeURLPath(rootDir, cfg.path)); err != nil {
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
	mux.HandleFunc("/_mdori/prism.css", servePrismCSS)
	mux.HandleFunc("/_mdori/prism.js", servePrismJS)
	mux.HandleFunc("/_mdori/", http.NotFound)
	mux.HandleFunc("/", a.serveDocument)
	mux.HandleFunc("/events", a.serveEvents)
	return mux
}

func (a *app) serveDocument(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/events" {
		a.serveEvents(w, r)
		return
	}

	filePath, err := a.resolveRequestPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !isMarkdownPath(filePath) {
		if err := a.serveStaticFile(w, r, filePath); err != nil {
			http.NotFound(w, r)
		}
		return
	}

	source, err := a.readMarkdownFile(filePath)
	if err != nil {
		http.Error(w, "failed to read markdown file", http.StatusInternalServerError)
		return
	}

	rendered, err := a.renderer.render(source)
	if err != nil {
		http.Error(w, "failed to render markdown", http.StatusInternalServerError)
		return
	}

	page, err := renderPage(pageTitle(filePath), rendered)
	if err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

func (a *app) readMarkdownFile(filePath string) ([]byte, error) {
	rootDir := a.rootDir
	if rootDir == "" {
		rootDir = filepath.Dir(a.sourcePath)
	}

	file, err := a.openFileWithinRoot(rootDir, filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return nil, errors.New("invalid path")
	}

	return io.ReadAll(file)
}

func (a *app) serveStaticFile(w http.ResponseWriter, r *http.Request, filePath string) error {
	rootDir := a.rootDir
	if rootDir == "" {
		rootDir = filepath.Dir(a.sourcePath)
	}

	file, err := a.openFileWithinRoot(rootDir, filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return errors.New("invalid path")
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}

func (a *app) openFileWithinRoot(rootDir, filePath string) (*os.File, error) {
	rel, err := filepath.Rel(rootDir, filePath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("invalid path")
	}

	if a.root != nil {
		return a.root.Open(rel)
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	return root.Open(rel)
}

func (a *app) resolveRequestPath(urlPath string) (string, error) {
	if urlPath == "/" {
		return "", errors.New("invalid path")
	}

	rootDir := a.rootDir
	if rootDir == "" {
		rootDir = filepath.Dir(a.sourcePath)
	}

	cleanPath := path.Clean("/" + urlPath)
	relURLPath := strings.TrimPrefix(cleanPath, "/")
	if relURLPath == "" || strings.HasPrefix(relURLPath, "../") {
		return "", errors.New("invalid path")
	}

	relPath := filepath.FromSlash(relURLPath)
	if filepath.IsAbs(relPath) {
		return "", errors.New("invalid path")
	}

	filePath := filepath.Join(rootDir, relPath)
	if !pathWithinRoot(rootDir, filePath) {
		return "", errors.New("invalid path")
	}

	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return "", errors.New("invalid path")
	}

	return filePath, nil
}

func rootRelativeURLPath(rootDir, filePath string) string {
	rel, err := filepath.Rel(rootDir, filePath)
	if err != nil || rel == "." || !pathWithinRoot(rootDir, filePath) {
		return "/"
	}

	return "/" + path.Clean(filepath.ToSlash(rel))
}

func pathWithinRoot(rootDir, filePath string) bool {
	resolvedRoot, err := filepath.EvalSymlinks(rootDir)
	if err == nil {
		rootDir = resolvedRoot
	}

	resolvedFile, err := filepath.EvalSymlinks(filePath)
	if err == nil {
		filePath = resolvedFile
	}

	rel, err := filepath.Rel(rootDir, filePath)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isMarkdownPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
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
