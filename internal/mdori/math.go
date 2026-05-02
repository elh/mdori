package mdori

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var kindMathInline = ast.NewNodeKind("MathInline")
var kindMathBlock = ast.NewNodeKind("MathBlock")

type mathInline struct {
	ast.BaseInline
	value []byte
}

func (n *mathInline) Kind() ast.NodeKind {
	return kindMathInline
}

func (n *mathInline) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type mathBlock struct {
	ast.BaseBlock
}

func (n *mathBlock) Kind() ast.NodeKind {
	return kindMathBlock
}

func (n *mathBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type mathInlineParser struct{}

func (p mathInlineParser) Trigger() []byte {
	return []byte{'$'}
}

func (p mathInlineParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	if len(line) < 3 || line[0] != '$' || line[1] == '$' || util.IsSpace(line[1]) {
		return nil
	}
	if segment.Start > 0 && reader.Source()[segment.Start-1] == '\\' {
		return nil
	}

	for i := 1; i < len(line); i++ {
		if line[i] != '$' || isEscaped(line, i) {
			continue
		}
		if i == 1 || util.IsSpace(line[i-1]) {
			return nil
		}

		reader.Advance(i + 1)
		return &mathInline{value: bytes.TrimSpace(line[1:i])}
	}

	return nil
}

type mathBlockParser struct{}

func (p mathBlockParser) Trigger() []byte {
	return []byte{'$'}
}

func (p mathBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || pos+2 > len(line) || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, parser.NoChildren
	}
	if !util.IsBlank(line[pos+2:]) {
		return nil, parser.NoChildren
	}

	reader.AdvanceToEOL()
	return &mathBlock{}, parser.NoChildren
}

func (p mathBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	pos := util.FirstNonSpacePosition(line)
	if pos >= 0 && pos+2 <= len(line) && bytes.HasPrefix(line[pos:], []byte("$$")) && util.IsBlank(line[pos+2:]) {
		reader.AdvanceToEOL()
		return parser.Close
	}

	node.Lines().Append(segment)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p mathBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (p mathBlockParser) CanInterruptParagraph() bool {
	return true
}

func (p mathBlockParser) CanAcceptIndentedLine() bool {
	return false
}

type mathRenderer struct{}

func (r mathRenderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMathInline, r.renderMathInline)
	reg.Register(kindMathBlock, r.renderMathBlock)
}

func (r mathRenderer) renderMathInline(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*mathInline)
	_, _ = w.WriteString(`<span class="mdori-math mdori-math-inline">`)
	_, _ = w.Write(util.EscapeHTML(n.value))
	_, _ = w.WriteString(`</span>`)
	return ast.WalkSkipChildren, nil
}

func (r mathRenderer) renderMathBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	_, _ = w.WriteString(`<div class="mdori-math mdori-math-display">`)
	writeCodeLines(w, source, node)
	_, _ = w.WriteString(`</div>` + "\n")
	return ast.WalkSkipChildren, nil
}

func isEscaped(line []byte, pos int) bool {
	count := 0
	for i := pos - 1; i >= 0 && line[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}
