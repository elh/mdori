package mdori

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type renderedDocument struct {
	HTML template.HTML
	TOC  []tocItem
}

type tocItem struct {
	Level int
	ID    string
	Text  string
}

type renderer struct {
	md goldmark.Markdown
}

func newRenderer() *renderer {
	return &renderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.Footnote),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
				parser.WithASTTransformers(util.Prioritized(githubTransformer{}, 400)),
			),
			goldmark.WithRendererOptions(gmrenderer.WithNodeRenderers(
				util.Prioritized(alertRenderer{}, 400),
				util.Prioritized(codeBlockRenderer{}, 400),
				util.Prioritized(safeHTMLRenderer{}, 400),
				util.Prioritized(tableRenderer{}, 400),
			)),
		),
	}
}

func (r *renderer) render(source []byte) (renderedDocument, error) {
	doc := r.md.Parser().Parse(gmtext.NewReader(source))

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, source, doc); err != nil {
		return renderedDocument{}, err
	}

	return renderedDocument{
		HTML: template.HTML(buf.String()),
		TOC:  collectTOC(doc, source),
	}, nil
}

func collectTOC(doc ast.Node, source []byte) []tocItem {
	var items []tocItem
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := node.(*ast.Heading)
		if !ok || heading.Level > 3 {
			return ast.WalkContinue, nil
		}

		idAttr, ok := heading.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}

		id := attributeString(idAttr)
		text := strings.Join(strings.Fields(string(heading.Text(source))), " ")
		if id == "" || text == "" {
			return ast.WalkContinue, nil
		}

		items = append(items, tocItem{
			Level: heading.Level,
			ID:    id,
			Text:  text,
		})

		return ast.WalkContinue, nil
	})

	return items
}

func attributeString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}
