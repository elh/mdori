package mdori

import (
	"bytes"
	"html/template"
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

	html := string(rendered.HTML)
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

func TestRendererWrapsTablesForHorizontalScrolling(t *testing.T) {
	t.Parallel()

	source := []byte("| Column | Value |\n| --- | --- |\n| long | value |\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	if !strings.Contains(html, `<div class="table-scroll">`+"\n<table>") {
		t.Fatalf("expected table scroll wrapper, got %q", html)
	}
	if !strings.Contains(html, "</table>\n</div>") {
		t.Fatalf("expected table scroll wrapper to close after table, got %q", html)
	}
}

func TestRendererBuildsTableOfContents(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\n## First Section\n\n### Child Section\n\n#### Too Deep\n\n## Second Section\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	expected := []tocItem{
		{Level: 1, ID: "title", Text: "Title"},
		{Level: 2, ID: "first-section", Text: "First Section"},
		{Level: 3, ID: "child-section", Text: "Child Section"},
		{Level: 2, ID: "second-section", Text: "Second Section"},
	}
	if len(rendered.TOC) != len(expected) {
		t.Fatalf("expected %d TOC items, got %#v", len(expected), rendered.TOC)
	}
	for i := range expected {
		if rendered.TOC[i] != expected[i] {
			t.Fatalf("expected TOC item %d to be %#v, got %#v", i, expected[i], rendered.TOC[i])
		}
	}
}

func TestRendererOmitsTrivialTableOfContents(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		"# Only Heading\n\nBody.\n",
		"# Title\n\n## Only Section\n\nBody.\n",
	} {
		rendered, err := newRenderer().render([]byte(source))
		if err != nil {
			t.Fatalf("render markdown: %v", err)
		}
		if len(rendered.TOC) != 0 {
			t.Fatalf("expected no TOC for %q, got %#v", source, rendered.TOC)
		}
	}
}

func TestRendererKeepsTableOfContentsForMultipleSections(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		"## First\n\n## Second\n",
		"# Title\n\n## First\n\n## Second\n",
	} {
		rendered, err := newRenderer().render([]byte(source))
		if err != nil {
			t.Fatalf("render markdown: %v", err)
		}
		if len(rendered.TOC) == 0 {
			t.Fatalf("expected TOC for %q", source)
		}
	}
}

func TestRendererUsesPlaintextClassForCodeBlocksWithoutLanguage(t *testing.T) {
	t.Parallel()

	source := []byte("Indented code:\n\n    package main\n\nFence without language:\n\n```\nplain\n```\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	if count := strings.Count(html, `<code class="language-plaintext">`); count != 2 {
		t.Fatalf("expected two plaintext code blocks, got %d in %q", count, html)
	}
}

func TestRendererSupportsMathExpressions(t *testing.T) {
	t.Parallel()

	source := []byte("Inline $E = mc^2$.\n\n$$\n<a_b>\n$$\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	for _, expected := range []string{
		`<span class="mdori-math mdori-math-inline">E = mc^2</span>`,
		`<div class="mdori-math mdori-math-display">&lt;a_b&gt;`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
	if !rendered.NeedsMath {
		t.Fatalf("expected math feature flag")
	}
	if rendered.NeedsMermaid {
		t.Fatalf("did not expect mermaid feature flag")
	}
}

func TestRendererDoesNotParseEscapedDollarOrCodeAsMath(t *testing.T) {
	t.Parallel()

	source := []byte(`Escaped \$x$ and code ` + "`$y$`" + ".\n\n```\n$z$\n```\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	if strings.Contains(html, "mdori-math") {
		t.Fatalf("did not expect math placeholders, got %q", html)
	}
	if rendered.NeedsMath {
		t.Fatalf("did not expect math feature flag")
	}
}

func TestRendererSupportsMermaidFences(t *testing.T) {
	t.Parallel()

	source := []byte("```mermaid\ngraph TD\n  A[<Start>] --> B\n```\n\n```go\npackage main\n```\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	for _, expected := range []string{
		`<div class="mdori-mermaid"><pre><code>graph TD`,
		`A[&lt;Start&gt;] --&gt; B`,
		`<pre><code class="language-go">package main`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
	if !rendered.NeedsMermaid {
		t.Fatalf("expected mermaid feature flag")
	}
	if rendered.NeedsMath {
		t.Fatalf("did not expect math feature flag")
	}
}

func TestRenderPageIncludesTableOfContents(t *testing.T) {
	t.Parallel()

	page, err := renderPage("doc", renderedDocument{
		HTML: template.HTML(`<h2 id="first">First</h2>`),
		TOC: []tocItem{
			{Level: 1, ID: "title", Text: "Title"},
			{Level: 2, ID: "first", Text: "First"},
			{Level: 3, ID: "child", Text: "Child"},
		},
	})
	if err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := string(page)
	for _, expected := range []string{
		`<nav class="toc" aria-label="Table of contents">`,
		`<a class="toc-link toc-level-1" href="#title">Title</a>`,
		`<a class="toc-link toc-level-2" href="#first">First</a>`,
		`<a class="toc-link toc-level-3" href="#child">Child</a>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
}

func TestRenderPageIncludesThemeSelector(t *testing.T) {
	t.Parallel()

	page, err := renderPage("doc", renderedDocument{HTML: template.HTML(`<h1 id="doc">Doc</h1>`)})
	if err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := string(page)
	for _, expected := range []string{
		`<select class="theme-select" aria-label="Color theme">`,
		`<option value="system">System</option>`,
		`<option value="light">Light</option>`,
		`<option value="dark">Dark</option>`,
		`localStorage.getItem("mdori-theme")`,
		`localStorage.setItem("mdori-theme", theme)`,
		`document.documentElement.dataset.theme = theme`,
		`html[data-theme="dark"]`,
		`@media (prefers-color-scheme: dark)`,
		`html[data-theme="system"]`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
}

func TestRenderPageConditionallyIncludesMathAssets(t *testing.T) {
	t.Parallel()

	page, err := renderPage("doc", renderedDocument{
		HTML:      template.HTML(`<p><span class="mdori-math mdori-math-inline">x</span></p>`),
		NeedsMath: true,
	})
	if err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := string(page)
	for _, expected := range []string{
		`<link rel="stylesheet" href="/_mdori/vendor/katex/katex.min.css">`,
		`<script defer src="/_mdori/vendor/katex/katex.min.js"></script>`,
		`<script defer src="/_mdori/mdori/math.js"></script>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
	if strings.Contains(html, "beautiful-mermaid") {
		t.Fatalf("did not expect mermaid assets in math page, got %q", html)
	}
}

func TestRenderPageConditionallyIncludesMermaidAssets(t *testing.T) {
	t.Parallel()

	page, err := renderPage("doc", renderedDocument{
		HTML:         template.HTML(`<div class="mdori-mermaid"></div>`),
		NeedsMermaid: true,
	})
	if err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := string(page)
	for _, expected := range []string{
		`<script defer src="/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js"></script>`,
		`<script defer src="/_mdori/mdori/mermaid.js"></script>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
	if strings.Contains(html, "katex") {
		t.Fatalf("did not expect math assets in mermaid page, got %q", html)
	}
}

func TestRenderPageOmitsOptionalAssetsForPlainMarkdown(t *testing.T) {
	t.Parallel()

	page, err := renderPage("doc", renderedDocument{HTML: template.HTML(`<p>plain</p>`)})
	if err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := string(page)
	for _, unexpected := range []string{"katex", "beautiful-mermaid", "/_mdori/mdori/math.js", "/_mdori/mdori/mermaid.js"} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("did not expect %q in output, got %q", unexpected, html)
		}
	}
}

func TestRendererDoesNotAllowRawHTMLByDefault(t *testing.T) {
	t.Parallel()

	rendered, err := newRenderer().render([]byte("<script>alert('x')</script>\n"))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	if strings.Contains(html, "<script>") {
		t.Fatalf("expected raw HTML to be omitted, got %q", html)
	}
}

func TestRendererAllowsSafeRawHTMLSubset(t *testing.T) {
	t.Parallel()

	source := []byte("<details open>\n<summary>More</summary>\n\n<sub>small</sub> <sup>high</sup> <ins>inserted</ins><br>\nPress <kbd>Enter</kbd>.\n</details>\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	for _, expected := range []string{
		"<details open>",
		"<summary>More</summary>",
		"<sub>small</sub>",
		"<sup>high</sup>",
		"<ins>inserted</ins>",
		"<br />",
		"<kbd>Enter</kbd>",
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

	html := string(rendered.HTML)
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

func TestRendererDropsUnsupportedSourceSrcAttribute(t *testing.T) {
	t.Parallel()

	source := []byte(`<picture><source src="unused.png" srcset="dark.png"><img src="light.png" alt="theme"></picture>` + "\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	if strings.Contains(html, `src="unused.png"`) {
		t.Fatalf("expected unsupported source src attribute to be omitted, got %q", html)
	}
	if !strings.Contains(html, `<source srcset="dark.png" />`) {
		t.Fatalf("expected source srcset attribute to remain, got %q", html)
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

		html := string(rendered.HTML)
		if strings.Contains(html, "<script>") || strings.Contains(html, "alert(1)") || strings.Contains(html, "benign comment") {
			t.Fatalf("expected raw HTML comment to be omitted for %q, got %q", source, html)
		}
		if !strings.Contains(html, "<!-- raw HTML omitted -->") {
			t.Fatalf("expected omitted raw HTML marker for %q, got %q", source, html)
		}
	}
}

func TestRendererSupportsGitHubAlerts(t *testing.T) {
	t.Parallel()

	source := []byte("> [!WARNING]\n> **Careful** with this.\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	for _, expected := range []string{
		`<div class="markdown-alert markdown-alert-warning">`,
		`<p class="markdown-alert-title">Warning</p>`,
		`<p><strong>Careful</strong> with this.</p>`,
		`</div>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
	if strings.Contains(html, "[!WARNING]") {
		t.Fatalf("expected alert marker to be omitted, got %q", html)
	}
}

func TestRendererBoldsDisplayOnlyGitHubReferences(t *testing.T) {
	t.Parallel()

	source := []byte("Refs: @octocat, @github/support, #123, owner/repo#789, GH-456, JIRA-123, user@example.com, and `@code #1`.\n")
	rendered, err := newRenderer().render(source)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	for _, expected := range []string{
		`<strong>@octocat</strong>`,
		`<strong>@github/support</strong>`,
		`<strong>#123</strong>`,
		`<strong>owner/repo#789</strong>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in output, got %q", expected, html)
		}
	}
	for _, unexpected := range []string{
		`<strong>GH-456</strong>`,
		`<strong>JIRA-123</strong>`,
		`user<strong>@example</strong>`,
		`<code><strong>@code</strong>`,
	} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("did not expect %q in output, got %q", unexpected, html)
		}
	}
}

func TestRendererDoesNotPromoteRawHTMLBlockText(t *testing.T) {
	t.Parallel()

	rendered, err := newRenderer().render([]byte("<script>\nalert(1)\n</script>\n"))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := string(rendered.HTML)
	if strings.Contains(html, "<script>") || strings.Contains(html, "alert(1)") {
		t.Fatalf("expected unsupported raw HTML block text to be omitted, got %q", html)
	}
}

func TestRendererGoldens(t *testing.T) {
	for _, name := range []string{"example", "simple_example", "math_example", "mermaid_example"} {
		t.Run(name, func(t *testing.T) {
			sourcePath := filepath.Join("testdata", name+".md")
			goldenPath := filepath.Join("testdata", name+".golden.html")

			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read %s fixture: %v", name, err)
			}

			rendered, err := newRenderer().render(source)
			if err != nil {
				t.Fatalf("render %s fixture: %v", name, err)
			}
			actual := []byte(rendered.HTML)

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
					t.Fatalf("update %s golden fixture: %v", name, err)
				}
			}

			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read %s golden fixture: %v", name, err)
			}

			if !bytes.Equal(actual, expected) {
				index := firstDiffIndex(actual, expected)
				t.Fatalf("rendered HTML differs from %s golden at byte %d\nexpected: %s\nactual:   %s\naccept with: UPDATE_GOLDEN=1 go test ./internal/mdori", name, index, excerptAt(expected, index), excerptAt(actual, index))
			}
		})
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
