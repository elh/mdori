package mdori

import (
	"bytes"
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

	first := getBody(t, server.URL+"/note.md")
	if !strings.Contains(first, "<h1 id=\"first\">First</h1>") {
		t.Fatalf("expected first response to contain initial heading, got %q", first)
	}

	if err := os.WriteFile(path, []byte("# Second\n"), 0o644); err != nil {
		t.Fatalf("write updated markdown: %v", err)
	}

	second := getBody(t, server.URL+"/note.md")
	if !strings.Contains(second, "<h1 id=\"second\">Second</h1>") {
		t.Fatalf("expected second response to contain updated heading, got %q", second)
	}
}

func TestPageTitleIncludesMarkdownExtension(t *testing.T) {
	t.Parallel()

	if got := pageTitle(filepath.Join("docs", "README.md")); got != "README.md" {
		t.Fatalf("expected title to include markdown extension, got %q", got)
	}
}

func TestServeDocumentFollowsRelativeMarkdownLinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs", "nested")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("make docs directory: %v", err)
	}

	indexPath := filepath.Join(docsDir, "index.md")
	linkedPath := filepath.Join(docsDir, "hello.md")
	readmePath := filepath.Join(dir, "README.md")

	if err := os.WriteFile(indexPath, []byte("[Local](./hello.md)\n\n[Parent](../../../README.md)\n\n[Root](/README.md)\n"), 0o644); err != nil {
		t.Fatalf("write index markdown: %v", err)
	}
	if err := os.WriteFile(linkedPath, []byte("# Local Document\n\n![Relative image](./image.png)\n"), 0o644); err != nil {
		t.Fatalf("write linked markdown: %v", err)
	}
	if err := os.WriteFile(readmePath, []byte("# Root Readme\n"), 0o644); err != nil {
		t.Fatalf("write readme markdown: %v", err)
	}

	handler := (&app{
		sourcePath: indexPath,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	server := httptest.NewServer(handler)
	defer server.Close()

	index := getBody(t, server.URL+"/docs/nested/index.md")
	if !strings.Contains(index, `<a href="./hello.md">Local</a>`) {
		t.Fatalf("expected index response to contain relative markdown link, got %q", index)
	}
	if !strings.Contains(index, `<a href="../../../README.md">Parent</a>`) {
		t.Fatalf("expected index response to contain parent markdown link, got %q", index)
	}
	if !strings.Contains(index, `<a href="/README.md">Root</a>`) {
		t.Fatalf("expected index response to contain root markdown link, got %q", index)
	}

	linked := getBody(t, server.URL+"/docs/nested/hello.md")
	if !strings.Contains(linked, `<h1 id="local-document">Local Document</h1>`) {
		t.Fatalf("expected linked markdown to render as HTML, got %q", linked)
	}
	if !strings.Contains(linked, `<img src="./image.png" alt="Relative image">`) {
		t.Fatalf("expected linked markdown to preserve relative image path, got %q", linked)
	}

	readme := getBody(t, server.URL+"/README.md")
	if !strings.Contains(readme, `<h1 id="root-readme">Root Readme</h1>`) {
		t.Fatalf("expected root markdown link to render as HTML, got %q", readme)
	}
}

func TestServeDocumentServesRelativeStaticFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	imagePath := filepath.Join(dir, "image.png")

	if err := os.WriteFile(path, []byte("![Relative image](./image.png)\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("fake image"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	server := httptest.NewServer(handler)
	defer server.Close()

	body := getBody(t, server.URL+"/index.md")
	if !strings.Contains(body, `<img src="./image.png" alt="Relative image">`) {
		t.Fatalf("expected rendered document to contain relative image path, got %q", body)
	}

	resp, err := http.Get(server.URL + "/image.png")
	if err != nil {
		t.Fatalf("get relative image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected image response status 200, got %d", resp.StatusCode)
	}

	image, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read image response: %v", err)
	}
	if string(image) != "fake image" {
		t.Fatalf("expected image body, got %q", image)
	}
}

func TestServeEmbeddedPrismAssets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	server := httptest.NewServer(handler)
	defer server.Close()

	page := getBody(t, server.URL+"/index.md")
	if !strings.Contains(page, `<script defer src="/_mdori/prism.js"></script>`) {
		t.Fatalf("expected rendered page to load prism js, got %q", page)
	}
	for _, unexpected := range []string{"katex", "beautiful-mermaid", "/_mdori/mdori/math.js", "/_mdori/mdori/mermaid.js"} {
		if strings.Contains(page, unexpected) {
			t.Fatalf("did not expect optional asset %q on plain page, got %q", unexpected, page)
		}
	}
	for _, expected := range []string{`.token.keyword`, `pre code[class*="language-"]`} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected rendered page to include prism token css %q, got %q", expected, page)
		}
	}

	resp, err := http.Get(server.URL + "/_mdori/prism.css")
	if err != nil {
		t.Fatalf("get prism css: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected prism css status 404, got %d", resp.StatusCode)
	}

	resp, err = http.Get(server.URL + "/_mdori/prism.js")
	if err != nil {
		t.Fatalf("get prism js: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected prism js status 200, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/javascript") {
		t.Fatalf("expected prism js content type, got %q", contentType)
	}
	js, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read prism js: %v", err)
	}
	for _, expected := range []string{"Prism.languages.go", "languages.bash", "languages.markdown"} {
		if !strings.Contains(string(js), expected) {
			t.Fatalf("expected prism js to include %q", expected)
		}
	}
}

func TestServeEmbeddedOptionalAssets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("Inline $x$.\n\n```mermaid\ngraph TD\n  A --> B\n```\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	server := httptest.NewServer(handler)
	defer server.Close()

	page := getBody(t, server.URL+"/index.md")
	for _, expected := range []string{
		`/_mdori/vendor/katex/katex.min.css`,
		`/_mdori/vendor/katex/katex.min.js`,
		`/_mdori/mdori/math.js`,
		`/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js`,
		`/_mdori/mdori/mermaid.js`,
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected page to include %q, got %q", expected, page)
		}
	}

	for _, tc := range []struct {
		path        string
		contentType string
		contains    string
	}{
		{"/_mdori/vendor/katex/katex.min.css", "text/css", ".katex"},
		{"/_mdori/vendor/katex/katex.min.js", "text/javascript", "katex"},
		{"/_mdori/vendor/katex/fonts/KaTeX_Main-Regular.woff2", "font/woff2", ""},
		{"/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js", "text/javascript", "mdoriBeautifulMermaid"},
		{"/_mdori/mdori/math.js", "text/javascript", "katex.render"},
		{"/_mdori/mdori/mermaid.js", "text/javascript", "renderMermaidSVG"},
	} {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("get %s: %v", tc.path, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s status 200, got %d", tc.path, resp.StatusCode)
		}
		if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, tc.contentType) {
			t.Fatalf("expected %s content type %q, got %q", tc.path, tc.contentType, contentType)
		}
		if tc.contains != "" && !strings.Contains(string(body), tc.contains) {
			t.Fatalf("expected %s to contain %q", tc.path, tc.contains)
		}
	}
}

func TestServeSimpleExampleDoesNotIncludeOptionalAssets(t *testing.T) {
	t.Parallel()

	server := serveTestdataMarkdown(t, "simple_example.md")
	page := getBody(t, server.URL+"/simple_example.md")
	for _, expected := range []string{
		`<h2 id="simple-example">Simple Example</h2>`,
		`This is a short paragraph with <code>inline code</code> and <strong>bold text</strong>.`,
		`<li>One <strong>important</strong> item</li>`,
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected simple page to include %q, got %q", expected, page)
		}
	}
	for _, unexpected := range []string{
		`/_mdori/vendor/katex/katex.min.css`,
		`/_mdori/vendor/katex/katex.min.js`,
		`/_mdori/mdori/math.js`,
		`/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js`,
		`/_mdori/mdori/mermaid.js`,
	} {
		if strings.Contains(page, unexpected) {
			t.Fatalf("did not expect simple page to include optional asset %q, got %q", unexpected, page)
		}
	}
}

func TestServeMathExampleDoesNotIncludeMermaidAssets(t *testing.T) {
	t.Parallel()

	server := serveTestdataMarkdown(t, "math_example.md")
	page := getBody(t, server.URL+"/math_example.md")
	for _, expected := range []string{
		`/_mdori/vendor/katex/katex.min.css`,
		`/_mdori/vendor/katex/katex.min.js`,
		`/_mdori/mdori/math.js`,
		`<span class="mdori-math mdori-math-inline">a^2 + b^2 = c^2</span>`,
		`<div class="mdori-math mdori-math-display">\sum_{n=1}^{3} n = 6`,
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected math page to include %q, got %q", expected, page)
		}
	}
	for _, unexpected := range []string{
		`/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js`,
		`/_mdori/mdori/mermaid.js`,
	} {
		if strings.Contains(page, unexpected) {
			t.Fatalf("did not expect math page to include %q, got %q", unexpected, page)
		}
	}
}

func TestServeMermaidExampleDoesNotIncludeMathAssets(t *testing.T) {
	t.Parallel()

	server := serveTestdataMarkdown(t, "mermaid_example.md")
	page := getBody(t, server.URL+"/mermaid_example.md")
	for _, expected := range []string{
		`/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js`,
		`/_mdori/mdori/mermaid.js`,
		`<div class="mdori-mermaid"><pre><code>flowchart TD`,
		`A[Start] --&gt; B[Done]`,
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected mermaid page to include %q, got %q", expected, page)
		}
	}
	for _, unexpected := range []string{
		`/_mdori/vendor/katex/katex.min.css`,
		`/_mdori/vendor/katex/katex.min.js`,
		`/_mdori/mdori/math.js`,
	} {
		if strings.Contains(page, unexpected) {
			t.Fatalf("did not expect mermaid page to include %q, got %q", unexpected, page)
		}
	}
}

func TestServeEmbeddedPrismPrefixDoesNotFallThroughToRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	internalDir := filepath.Join(dir, "_mdori")
	if err := os.Mkdir(internalDir, 0o755); err != nil {
		t.Fatalf("make internal-looking directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(internalDir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write internal-looking file: %v", err)
	}

	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/_mdori/secret.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected internal asset prefix status 404, got %d with body %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("expected internal asset prefix not to expose file body, got %q", rec.Body.String())
	}
}

func TestServeDocumentDoesNotRenderSourceAtRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected root response status 404, got %d with body %q", rec.Code, rec.Body.String())
	}
}

func TestRootRelativeURLPath(t *testing.T) {
	t.Parallel()

	got := rootRelativeURLPath(filepath.Join("repo"), filepath.Join("repo", "internal", "mdori", "testdata", "example.md"))
	want := "/internal/mdori/testdata/example.md"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestServeDocumentDoesNotServeFilesOutsideRootDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	if err := os.Mkdir(rootDir, 0o755); err != nil {
		t.Fatalf("make source directory: %v", err)
	}

	path := filepath.Join(rootDir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    rootDir,
		renderer:   newRenderer(),
	}).routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/%2e%2e/secret.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected traversal response status 404, got %d with body %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("expected traversal response not to expose file body, got %q", rec.Body.String())
	}
}

func TestServeDocumentDoesNotFollowSymlinksOutsideRootDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	outsideDir := filepath.Join(dir, "outside")
	if err := os.Mkdir(rootDir, 0o755); err != nil {
		t.Fatalf("make root directory: %v", err)
	}
	if err := os.Mkdir(outsideDir, 0o755); err != nil {
		t.Fatalf("make outside directory: %v", err)
	}

	path := filepath.Join(rootDir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(rootDir, "outside-link")); err != nil {
		t.Fatalf("make symlink: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    rootDir,
		renderer:   newRenderer(),
	}).routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/outside-link/secret.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected symlink escape response status 404, got %d with body %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("expected symlink escape response not to expose file body, got %q", rec.Body.String())
	}
}

func TestServeDocumentDoesNotServeSymlinkedFileOutsideRootDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	outsideDir := filepath.Join(dir, "outside")
	if err := os.Mkdir(rootDir, 0o755); err != nil {
		t.Fatalf("make root directory: %v", err)
	}
	if err := os.Mkdir(outsideDir, 0o755); err != nil {
		t.Fatalf("make outside directory: %v", err)
	}

	path := filepath.Join(rootDir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	secretPath := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if err := os.Symlink(secretPath, filepath.Join(rootDir, "secret-link.txt")); err != nil {
		t.Fatalf("make symlink: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    rootDir,
		renderer:   newRenderer(),
	}).routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/secret-link.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected symlink file response status 404, got %d with body %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("expected symlink file response not to expose file body, got %q", rec.Body.String())
	}
}

func TestServeDocumentDoesNotRenderSymlinkedMarkdownOutsideRootDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	outsideDir := filepath.Join(dir, "outside")
	if err := os.Mkdir(rootDir, 0o755); err != nil {
		t.Fatalf("make root directory: %v", err)
	}
	if err := os.Mkdir(outsideDir, 0o755); err != nil {
		t.Fatalf("make outside directory: %v", err)
	}

	path := filepath.Join(rootDir, "index.md")
	if err := os.WriteFile(path, []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	secretPath := filepath.Join(outsideDir, "secret.md")
	if err := os.WriteFile(secretPath, []byte("# Secret\n"), 0o644); err != nil {
		t.Fatalf("write secret markdown: %v", err)
	}
	if err := os.Symlink(secretPath, filepath.Join(rootDir, "secret-link.md")); err != nil {
		t.Fatalf("make symlink: %v", err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    rootDir,
		renderer:   newRenderer(),
	}).routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/secret-link.md", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected symlink markdown response status 404, got %d with body %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Secret") {
		t.Fatalf("expected symlink markdown response not to expose file body, got %q", rec.Body.String())
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

func TestParseArgsUsageMatchesFlagOutput(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	_, err := parseArgs(nil, &stderr)
	if err == nil {
		t.Fatal("expected parse args to fail without markdown file")
	}

	output := stderr.String()
	for _, expected := range []string{
		"usage: mdori [-addr host:port] [-no-open] [-o output.html] [-preview] <markdown-file>",
		"  -addr string",
		"  -no-open",
		"  -o string",
		"  -preview",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected usage output to contain %q, got %q", expected, output)
		}
	}
	if strings.Contains(output, "--addr") || strings.Contains(output, "--no-open") {
		t.Fatalf("expected usage output to use single-dash flag style, got %q", output)
	}
}

func TestRunReturnsWhenContextCancels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = Run(ctx, []string{"--no-open", path}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("expected clean shutdown on canceled context, got %v", err)
	}
}

func TestRunPrintsRenderedURLAndStopMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout bytes.Buffer
	err = Run(ctx, []string{"--no-open", path}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("expected clean shutdown on canceled context, got %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{"Serving at http://", "/README.md", "Press Ctrl-C to stop."} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected stdout to contain %q, got %q", expected, output)
		}
	}
}

func TestRunWritesStandaloneOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "math.md")
	outputPath := filepath.Join(dir, "math.html")
	if err := os.WriteFile(path, []byte("Inline $x$.\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	if err := Run(context.Background(), []string{"-o", outputPath, path}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run output mode: %v", err)
	}

	html, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	page := string(html)
	for _, expected := range []string{
		`<span class="mdori-math mdori-math-inline">x</span>`,
		`<style>`,
		`data:font/woff2;base64,`,
		`<script>`,
		`katex.render`,
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected standalone output to contain %q", expected)
		}
	}
	for _, unexpected := range []string{
		`/_mdori/`,
		`new EventSource`,
	} {
		if strings.Contains(page, unexpected) {
			t.Fatalf("did not expect standalone output to contain %q", unexpected)
		}
	}
}

func TestRenderPreviewFileWritesTemporaryStandaloneOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram.md")
	if err := os.WriteFile(path, []byte("```mermaid\ngraph TD\n  A --> B\n```\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	previewPath, err := renderPreviewFile(path)
	if err != nil {
		t.Fatalf("render preview file: %v", err)
	}

	html, err := os.ReadFile(previewPath)
	if err != nil {
		t.Fatalf("read preview file: %v", err)
	}
	page := string(html)
	for _, expected := range []string{
		`<div class="mdori-mermaid"><pre><code>graph TD`,
		`mdoriBeautifulMermaid`,
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("expected preview output to contain %q", expected)
		}
	}
	if strings.Contains(page, `/_mdori/`) || strings.Contains(page, `new EventSource`) {
		t.Fatalf("expected preview output to be standalone, got %q", page)
	}
}

func TestRunRejectsMarkdownFileOutsideWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	outsideDir := filepath.Join(dir, "outside")
	if err := os.Mkdir(rootDir, 0o755); err != nil {
		t.Fatalf("make root directory: %v", err)
	}
	if err := os.Mkdir(outsideDir, 0o755); err != nil {
		t.Fatalf("make outside directory: %v", err)
	}

	path := filepath.Join(outsideDir, "README.md")
	if err := os.WriteFile(path, []byte("# Outside\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	err = Run(context.Background(), []string{"--no-open", path}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected Run to reject source outside working directory")
	}
	if !strings.Contains(err.Error(), "outside serving root") {
		t.Fatalf("expected outside serving root error, got %v", err)
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

func serveTestdataMarkdown(t *testing.T, name string) *httptest.Server {
	t.Helper()

	source, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}

	handler := (&app{
		sourcePath: path,
		rootDir:    dir,
		renderer:   newRenderer(),
	}).routes()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}
