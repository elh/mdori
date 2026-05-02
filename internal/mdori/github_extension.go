package mdori

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const alertAttr = "mdori-alert"

var (
	alertMarkerPattern = regexp.MustCompile(`^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\](?:\r?\n)?`)
	referencePattern   = regexp.MustCompile(`(^|[^[:alnum:]_/@])([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+#[0-9]+|@[A-Za-z0-9](?:[A-Za-z0-9-]{0,38}[A-Za-z0-9])?(?:/[A-Za-z0-9](?:[A-Za-z0-9-]{0,38}[A-Za-z0-9])?)?|#[0-9]+)`)
)

type githubTransformer struct{}

var _ parser.ASTTransformer = githubTransformer{}

func (t githubTransformer) Transform(doc *ast.Document, reader gmtext.Reader, _ parser.Context) {
	source := reader.Source()
	var referenceTextNodes []*ast.Text

	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if blockquote, ok := node.(*ast.Blockquote); ok {
			transformAlert(blockquote, source)
			return ast.WalkContinue, nil
		}

		if textNode, ok := node.(*ast.Text); ok && !hasAncestor(node, ast.KindCodeSpan) {
			if transformAlertText(textNode, source) {
				return ast.WalkContinue, nil
			}
			referenceTextNodes = append(referenceTextNodes, textNode)
		}
		if stringNode, ok := node.(*ast.String); ok && !hasAncestor(node, ast.KindCodeSpan) {
			transformAlertString(stringNode)
		}

		return ast.WalkContinue, nil
	})

	for _, textNode := range referenceTextNodes {
		if textNode.Parent() != nil {
			transformReferences(textNode, source)
		}
	}
}

type alertRenderer struct{}

func (r alertRenderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
}

func (r alertRenderer) renderBlockquote(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	alert, ok := node.AttributeString(alertAttr)
	if !ok {
		if entering {
			_, _ = w.WriteString("<blockquote>\n")
		} else {
			_, _ = w.WriteString("</blockquote>\n")
		}
		return ast.WalkContinue, nil
	}

	kind := alert.(string)
	if entering {
		_, _ = w.WriteString(`<div class="markdown-alert markdown-alert-`)
		_, _ = w.WriteString(kind)
		_, _ = w.WriteString("\">\n")
		_, _ = w.WriteString(`<p class="markdown-alert-title">`)
		_, _ = w.WriteString(alertTitle(kind))
		_, _ = w.WriteString("</p>\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

func transformAlertText(textNode *ast.Text, source []byte) bool {
	blockquote := ancestorBlockquote(textNode)
	if blockquote == nil || firstTextLikeChild(blockquote) != textNode {
		return false
	}

	value := string(textNode.Value(source))
	match := alertMarkerPattern.FindStringSubmatchIndex(value)
	if match == nil || match[0] != 0 {
		return false
	}

	kind := strings.ToLower(value[match[2]:match[3]])
	blockquote.SetAttributeString(alertAttr, kind)

	markerLen := match[1]
	if markerLen >= len(value) {
		parent := textNode.Parent()
		parent.RemoveChild(parent, textNode)
		return true
	}

	textNode.Segment = gmtext.NewSegment(textNode.Segment.Start+markerLen, textNode.Segment.Stop)
	return true
}

func transformAlertString(stringNode *ast.String) bool {
	blockquote := ancestorBlockquote(stringNode)
	if blockquote == nil || firstTextLikeChild(blockquote) != stringNode {
		return false
	}

	value := string(stringNode.Value)
	match := alertMarkerPattern.FindStringSubmatchIndex(value)
	if match == nil || match[0] != 0 {
		return false
	}

	kind := strings.ToLower(value[match[2]:match[3]])
	blockquote.SetAttributeString(alertAttr, kind)

	markerLen := match[1]
	if markerLen >= len(value) {
		parent := stringNode.Parent()
		parent.RemoveChild(parent, stringNode)
		return true
	}

	stringNode.Value = stringNode.Value[markerLen:]
	return true
}

func transformAlert(blockquote *ast.Blockquote, source []byte) {
	paragraph, ok := blockquote.FirstChild().(*ast.Paragraph)
	if !ok {
		return
	}

	value := string(paragraph.Text(source))
	match := alertMarkerPattern.FindStringSubmatchIndex(value)
	if match == nil || match[0] != 0 {
		return
	}

	kind := strings.ToLower(value[match[2]:match[3]])
	blockquote.SetAttributeString(alertAttr, kind)
	consumeLeadingText(paragraph, source, match[1])
}

func transformReferences(textNode *ast.Text, source []byte) {
	if textNode.IsRaw() {
		return
	}

	value := string(textNode.Value(source))
	matches := referencePattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return
	}

	parent := textNode.Parent()
	if parent == nil {
		return
	}

	cursor := 0
	for _, match := range matches {
		prefixStart, prefixEnd := match[2], match[3]
		refStart, refEnd := match[4], match[5]

		appendTextSegmentBefore(parent, textNode, textNode, cursor, prefixEnd)
		if prefixStart != prefixEnd {
			cursor = prefixEnd
		}

		strong := ast.NewEmphasis(2)
		strong.AppendChild(strong, ast.NewTextSegment(segmentForTextRange(textNode, refStart, refEnd)))
		parent.InsertBefore(parent, textNode, strong)
		cursor = refEnd
	}

	appendTextSegmentBefore(parent, textNode, textNode, cursor, len(value))
	parent.RemoveChild(parent, textNode)
}

func appendTextSegmentBefore(parent, before ast.Node, original *ast.Text, start, stop int) {
	if start >= stop {
		return
	}

	node := ast.NewTextSegment(segmentForTextRange(original, start, stop))
	if stop == original.Segment.Stop-original.Segment.Start {
		node.SetSoftLineBreak(original.SoftLineBreak())
		node.SetHardLineBreak(original.HardLineBreak())
	}
	parent.InsertBefore(parent, before, node)
}

func segmentForTextRange(original *ast.Text, start, stop int) gmtext.Segment {
	return gmtext.NewSegment(original.Segment.Start+start, original.Segment.Start+stop)
}

func hasAncestor(node ast.Node, kind ast.NodeKind) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Kind() == kind {
			return true
		}
	}
	return false
}

func ancestorBlockquote(node ast.Node) *ast.Blockquote {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if blockquote, ok := parent.(*ast.Blockquote); ok {
			return blockquote
		}
	}
	return nil
}

func firstTextLikeChild(parent ast.Node) ast.Node {
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *ast.Text, *ast.String:
			return child
		}
		if found := firstTextLikeChild(child); found != nil {
			return found
		}
	}
	return nil
}

func consumeLeadingText(parent ast.Node, source []byte, remaining int) {
	for child := parent.FirstChild(); child != nil && remaining > 0; {
		next := child.NextSibling()
		textLen, lineBreakLen := textLikeLen(child, source)
		totalLen := textLen + lineBreakLen
		if totalLen == 0 {
			child = next
			continue
		}

		if remaining >= totalLen {
			parent.RemoveChild(parent, child)
			remaining -= totalLen
			child = next
			continue
		}

		if remaining < textLen {
			trimTextLike(child, remaining)
			return
		}

		parent.RemoveChild(parent, child)
		return
	}
}

func textLikeLen(node ast.Node, source []byte) (int, int) {
	switch n := node.(type) {
	case *ast.Text:
		lineBreakLen := 0
		if n.SoftLineBreak() || n.HardLineBreak() {
			lineBreakLen = 1
		}
		return len(n.Value(source)), lineBreakLen
	case *ast.String:
		return len(n.Value), 0
	default:
		return 0, 0
	}
}

func trimTextLike(node ast.Node, n int) {
	switch textNode := node.(type) {
	case *ast.Text:
		textNode.Segment = gmtext.NewSegment(textNode.Segment.Start+n, textNode.Segment.Stop)
	case *ast.String:
		textNode.Value = textNode.Value[n:]
	}
}

func alertTitle(kind string) string {
	switch kind {
	case "note":
		return "Note"
	case "tip":
		return "Tip"
	case "important":
		return "Important"
	case "warning":
		return "Warning"
	case "caution":
		return "Caution"
	default:
		return kind
	}
}
