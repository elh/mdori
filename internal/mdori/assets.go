package mdori

import (
	"bytes"
	_ "embed"
	"net/http"
)

// Prism 1.30.0, vendored from prismjs.com/unpkg.
//
// Keep this list intentionally small. Goldmark already emits language-* classes;
// Prism only needs to style the common languages we bundle here.
//
//go:embed assets/prism.css
var prismCSS []byte

//go:embed assets/prism-core.js
var prismCoreJS []byte

//go:embed assets/prism-markup.js
var prismMarkupJS []byte

//go:embed assets/prism-css.js
var prismCSSJS []byte

//go:embed assets/prism-clike.js
var prismCLikeJS []byte

//go:embed assets/prism-javascript.js
var prismJavaScriptJS []byte

//go:embed assets/prism-go.js
var prismGoJS []byte

//go:embed assets/prism-bash.js
var prismBashJS []byte

//go:embed assets/prism-json.js
var prismJSONJS []byte

//go:embed assets/prism-markdown.js
var prismMarkdownJS []byte

var prismJS = bytes.Join([][]byte{
	prismCoreJS,
	prismMarkupJS,
	prismCSSJS,
	prismCLikeJS,
	prismJavaScriptJS,
	prismGoJS,
	prismBashJS,
	prismJSONJS,
	prismMarkdownJS,
}, []byte("\n"))

func servePrismCSS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(prismCSS)
}

func servePrismJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(prismJS)
}
