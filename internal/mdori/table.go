package mdori

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type tableRenderer struct{}

func (r tableRenderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(east.KindTable, r.renderTable)
}

func (r tableRenderer) renderTable(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="table-scroll">` + "\n")
		_, _ = w.WriteString("<table")
		if node.Attributes() != nil {
			html.RenderAttributes(w, node, extension.TableAttributeFilter)
		}
		_, _ = w.WriteString(">\n")
	} else {
		_, _ = w.WriteString("</table>\n")
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
