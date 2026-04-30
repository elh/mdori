package mdori

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeDocumentReloadsSourceOnEachRequest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")

	if err := os.WriteFile(path, []byte("# First\n"), 0o644); err != nil {
		t.Fatalf("write initial markdown: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		renderer:   newRenderer(),
	}).routes()

	server := httptest.NewServer(handler)
	defer server.Close()

	first := getBody(t, server.URL+"/")
	if !strings.Contains(first, "<h1 id=\"first\">First</h1>") {
		t.Fatalf("expected first response to contain initial heading, got %q", first)
	}

	if err := os.WriteFile(path, []byte("# Second\n"), 0o644); err != nil {
		t.Fatalf("write updated markdown: %v", err)
	}

	second := getBody(t, server.URL+"/")
	if !strings.Contains(second, "<h1 id=\"second\">Second</h1>") {
		t.Fatalf("expected second response to contain updated heading, got %q", second)
	}
}

func TestParseArgsDefaultsAndFileResolution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	cfg, err := parseArgs([]string{path}, io.Discard)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	if cfg.addr != "127.0.0.1:0" {
		t.Fatalf("unexpected default addr %q", cfg.addr)
	}

	if !cfg.openBrowser {
		t.Fatal("expected browser opening to be enabled by default")
	}

	if !filepath.IsAbs(cfg.path) {
		t.Fatalf("expected absolute path, got %q", cfg.path)
	}
}

func TestRunReturnsWhenContextCancels(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Run(ctx, []string{"--no-open", path}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("expected clean shutdown on canceled context, got %v", err)
	}
}

func getBody(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	return string(body)
}
