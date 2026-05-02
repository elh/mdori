package mdori

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestRendererUsesPlaintextClassForCodeBlocksWithoutLanguage(t *testing.T) {
	t.Parallel()

	source := []byte("Indented code:\n\n    package main\n\nFence without language:\n\n```\nplain\n```\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered)
	if count := strings.Count(html, `<code class="language-plaintext">`); count != 2 {
		t.Fatalf("expected two plaintext code blocks, got %d in %q", count, html)
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

func TestRendererAllowsSafeRawHTMLSubset(t *testing.T) {
	t.Parallel()

	source := []byte("<details open>\n<summary>More</summary>\n\n<sub>small</sub> <sup>high</sup> <ins>inserted</ins>\n</details>\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered)
	for _, expected := range []string{
		"<details open>",
		"<summary>More</summary>",
		"<sub>small</sub>",
		"<sup>high</sup>",
		"<ins>inserted</ins>",
		"</details>",
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
}

func TestRendererDropsUnsafeRawHTMLAttributes(t *testing.T) {
	t.Parallel()

	source := []byte(`<A HREF="javascript:alert(1)" onclick="alert(2)" title="ok">link</A>` + "\n\n" + `<img src="data:text/html;base64,PHNjcmlwdA==" alt="bad" onerror="alert(3)">` + "\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered)
	for _, unexpected := range []string{"javascript:", "data:text/html", "onclick", "onerror", "alert("} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("expected unsafe attribute content %q to be omitted, got %q", unexpected, html)
		}
	}
	if !strings.Contains(html, `<a title="ok">link</a>`) {
		t.Fatalf("expected safe anchor attributes to remain, got %q", html)
	}
	if !strings.Contains(html, `<img alt="bad" />`) {
		t.Fatalf("expected safe image attributes to remain, got %q", html)
	}
}

func TestRendererDropsUnsafeHTMLCommentTerminators(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		`<!-- benign comment -->`,
		`<!-- --><script>alert(1)</script><!-- -->`,
		`<!-- --!><script>alert(1)</script><!-- -->`,
	} {
		rendered, err := newRenderer().render([]byte(source + "\n"))
		if err != nil {
			t.Fatalf("render markdown: %v", err)
		}

		html := string(rendered)
		if strings.Contains(html, "<script>") || strings.Contains(html, "alert(1)") || strings.Contains(html, "benign comment") {
			t.Fatalf("expected raw HTML comment to be omitted for %q, got %q", source, html)
		}
		if !strings.Contains(html, "<!-- raw HTML omitted -->") {
			t.Fatalf("expected omitted raw HTML marker for %q, got %q", source, html)
		}
	}
}

func TestRendererDoesNotPromoteRawHTMLBlockText(t *testing.T) {
	t.Parallel()

	rendered, err := newRenderer().render([]byte("<script>\nalert(1)\n</script>\n"))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered)
	if strings.Contains(html, "<script>") || strings.Contains(html, "alert(1)") {
		t.Fatalf("expected unsupported raw HTML block text to be omitted, got %q", html)
	}
}

func TestRendererExampleGolden(t *testing.T) {
	sourcePath := filepath.Join("testdata", "example.md")
	goldenPath := filepath.Join("testdata", "example.golden.html")

	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read example fixture: %v", err)
	}

	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render example fixture: %v", err)
	}
	actual := []byte(rendered)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
			t.Fatalf("update golden fixture: %v", err)
		}
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	if !bytes.Equal(actual, expected) {
		index := firstDiffIndex(actual, expected)
		t.Fatalf("rendered HTML differs from golden at byte %d\nexpected: %s\nactual:   %s\naccept with: UPDATE_GOLDEN=1 go test ./internal/mdori", index, excerptAt(expected, index), excerptAt(actual, index))
	}
}

func firstDiffIndex(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}

	if len(a) != len(b) {
		if len(a) < len(b) {
			return len(a)
		}
		return len(b)
	}

	return -1
}

func excerptAt(b []byte, index int) string {
	if index < 0 {
		return ""
	}

	start := index - 40
	if start < 0 {
		start = 0
	}

	end := index + 80
	if end > len(b) {
		end = len(b)
	}

	return strings.ReplaceAll(string(b[start:end]), "\n", `\n`)
}
