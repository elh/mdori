package mdori

import (
	"strings"
	"testing"
)

func TestRendererSupportsGitHubFlavoredMarkdown(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\n- [x] done\n\n| A | B |\n| - | - |\n| 1 | 2 |\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered)
	if !strings.Contains(html, "<h1 id=\"title\">Title</h1>") {
		t.Fatalf("expected heading id in output, got %q", html)
	}

	if !strings.Contains(html, "<input checked=\"\" disabled=\"\" type=\"checkbox\">") {
		t.Fatalf("expected task list checkbox in output, got %q", html)
	}

	if !strings.Contains(html, "<table>") {
		t.Fatalf("expected GFM table in output, got %q", html)
	}
}

func TestRendererDoesNotAllowRawHTMLByDefault(t *testing.T) {
	t.Parallel()

	rendered, err := newRenderer().render([]byte("<script>alert('x')</script>\n"))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered)
	if strings.Contains(html, "<script>") {
		t.Fatalf("expected raw HTML to be omitted, got %q", html)
	}
}
