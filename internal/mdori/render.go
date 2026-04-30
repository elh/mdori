package mdori

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type renderer struct {
	md goldmark.Markdown
}

func newRenderer() *renderer {
	return &renderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
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
