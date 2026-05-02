package mdori

import (
	"html"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type codeBlockRenderer struct{}

func (r codeBlockRenderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r codeBlockRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<pre><code class="language-plaintext">`)
		writeCodeLines(w, source, node)
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}

	return ast.WalkContinue, nil
}

func (r codeBlockRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		language := n.Language(source)
		if len(language) == 0 {
			language = []byte("plaintext")
		}

		if isMermaidLanguage(language) {
			_, _ = w.WriteString(`<div class="mdori-mermaid"><pre><code>`)
			writeCodeLines(w, source, node)
			return ast.WalkContinue, nil
		}

		_, _ = w.WriteString(`<pre><code class="language-`)
		_, _ = w.WriteString(html.EscapeString(string(language)))
		_, _ = w.WriteString(`">`)
		writeCodeLines(w, source, node)
	} else {
		language := n.Language(source)
		if isMermaidLanguage(language) {
			_, _ = w.WriteString("</code></pre></div>\n")
		} else {
			_, _ = w.WriteString("</code></pre>\n")
		}
	}

	return ast.WalkContinue, nil
}

func writeCodeLines(w util.BufWriter, source []byte, node ast.Node) {
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		_, _ = w.Write(util.EscapeHTML(line.Value(source)))
	}
}

func isMermaidLanguage(language []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(language)), "mermaid")
}
