package mdori

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type renderer struct {
	md goldmark.Markdown
}

func newRenderer() *renderer {
	return &renderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.Footnote),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(gmrenderer.WithNodeRenderers(
				util.Prioritized(codeBlockRenderer{}, 400),
				util.Prioritized(safeHTMLRenderer{}, 400),
			)),
		),
	}
}

func (r *renderer) render(source []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}
