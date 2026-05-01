package mdori

import (
	"bytes"
	"html"
	"io"
	"net/url"
	"slices"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
)

type safeHTMLRenderer struct{}

func (r safeHTMLRenderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
}

func (r safeHTMLRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)
	if entering {
		for i := range n.Lines().Len() {
			segment := n.Lines().At(i)
			line := segment.Value(source)
			_, _ = w.Write(sanitizeHTMLLine(line))
		}
	} else if n.HasClosure() {
		_, _ = w.Write(sanitizeHTMLLine(n.ClosureLine.Value(source)))
	}

	return ast.WalkContinue, nil
}

func (r safeHTMLRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	n := node.(*ast.RawHTML)
	for i := range n.Segments.Len() {
		segment := n.Segments.At(i)
		_, _ = w.Write(sanitizeHTMLLine(segment.Value(source)))
	}

	return ast.WalkSkipChildren, nil
}

func sanitizeHTMLLine(raw []byte) []byte {
	lineEnd := []byte{}
	trimmedRight := bytes.TrimRight(raw, "\r\n")
	if len(trimmedRight) < len(raw) {
		lineEnd = raw[len(trimmedRight):]
	}

	prefixLen := len(trimmedRight) - len(bytes.TrimLeftFunc(trimmedRight, unicode.IsSpace))
	suffixLen := len(trimmedRight) - len(bytes.TrimRightFunc(trimmedRight, unicode.IsSpace))
	prefix := trimmedRight[:prefixLen]
	suffix := trimmedRight[len(trimmedRight)-suffixLen:]
	token := strings.TrimSpace(string(trimmedRight))

	safe, ok := sanitizeHTMLToken(token)
	if !ok {
		safe = "<!-- raw HTML omitted -->"
	}

	out := make([]byte, 0, len(prefix)+len(safe)+len(suffix)+len(lineEnd))
	out = append(out, prefix...)
	out = append(out, safe...)
	out = append(out, suffix...)
	out = append(out, lineEnd...)
	return out
}

func sanitizeHTMLToken(token string) (string, bool) {
	var b strings.Builder
	z := xhtml.NewTokenizer(strings.NewReader(token))
	wroteToken := false
	wroteTag := false

	for {
		tokenType := z.Next()
		switch tokenType {
		case xhtml.ErrorToken:
			if z.Err() != nil && z.Err() != io.EOF {
				return "", false
			}
			return b.String(), wroteToken && wroteTag

		case xhtml.TextToken:
			text := string(z.Text())
			if text == "" {
				continue
			}
			b.WriteString(html.EscapeString(text))
			wroteToken = true

		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			tag := strings.ToLower(string(tagName))
			if tag == "" || !allowedHTMLTags[tag] {
				return "", false
			}

			var attrs []htmlAttr
			seen := map[string]bool{}
			for hasAttr {
				key, value, more := z.TagAttr()
				name := strings.ToLower(string(key))
				if !seen[name] {
					seen[name] = true
					if sanitized, ok := sanitizeHTMLAttr(tag, name, string(value)); ok {
						attrs = append(attrs, htmlAttr{name: name, value: sanitized})
					}
				}
				hasAttr = more
			}

			b.WriteString(renderHTMLStartTag(tag, attrs, tokenType == xhtml.SelfClosingTagToken || voidHTMLTags[tag]))
			wroteToken = true
			wroteTag = true

		case xhtml.EndTagToken:
			tagName, _ := z.TagName()
			tag := strings.ToLower(string(tagName))
			if tag == "" || !allowedHTMLTags[tag] || voidHTMLTags[tag] {
				return "", false
			}
			b.WriteString("</")
			b.WriteString(tag)
			b.WriteByte('>')
			wroteToken = true
			wroteTag = true

		case xhtml.CommentToken, xhtml.DoctypeToken:
			return "", false
		}
	}
}

type htmlAttr struct {
	name  string
	value string
}

func sanitizeHTMLAttr(tag, name, value string) (string, bool) {
	if !slices.Contains(allowedHTMLAttrs[tag], name) {
		return "", false
	}

	if booleanHTMLAttrs[name] {
		if value == "" || strings.EqualFold(value, name) {
			return "", true
		}
		return "", false
	}

	switch name {
	case "href":
		return value, isSafeURL(value, true)
	case "src":
		return value, isSafeURL(value, false)
	case "srcset":
		return value, isSafeSrcset(value)
	default:
		return value, true
	}
}

func renderHTMLStartTag(tag string, attrs []htmlAttr, selfClosing bool) string {
	var b strings.Builder
	b.WriteByte('<')
	b.WriteString(tag)
	for _, attr := range attrs {
		b.WriteByte(' ')
		b.WriteString(attr.name)
		if attr.value != "" || !booleanHTMLAttrs[attr.name] {
			b.WriteString(`="`)
			b.WriteString(html.EscapeString(attr.value))
			b.WriteByte('"')
		}
	}
	if selfClosing {
		b.WriteString(" />")
	} else {
		b.WriteByte('>')
	}
	return b.String()
}

func isSafeURL(value string, allowMailto bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		return true
	}

	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if u.Scheme == "" {
		return !strings.Contains(value, ":")
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	case "mailto":
		return allowMailto
	default:
		return false
	}
}

func isSafeSrcset(value string) bool {
	for _, candidate := range strings.Split(value, ",") {
		fields := strings.Fields(strings.TrimSpace(candidate))
		if len(fields) == 0 || !isSafeURL(fields[0], false) {
			return false
		}
	}
	return true
}

var allowedHTMLTags = map[string]bool{
	"a":       true,
	"details": true,
	"img":     true,
	"ins":     true,
	"picture": true,
	"source":  true,
	"sub":     true,
	"summary": true,
	"sup":     true,
}

var voidHTMLTags = map[string]bool{
	"img":    true,
	"source": true,
}

var booleanHTMLAttrs = map[string]bool{
	"open": true,
}

var allowedHTMLAttrs = map[string][]string{
	"a":       {"href", "id", "name", "title"},
	"details": {"open"},
	"img":     {"alt", "height", "loading", "src", "srcset", "title", "width"},
	"ins":     {},
	"picture": {},
	"source":  {"media", "sizes", "src", "srcset", "type"},
	"sub":     {},
	"summary": {},
	"sup":     {},
}
