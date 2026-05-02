package mdori

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Prism 1.30.0 JavaScript, vendored from prismjs.com/unpkg.
//
// Keep this list intentionally small. Goldmark already emits language-* classes;
// Prism only needs the common languages we bundle here.

//go:embed assets/third_party/prism/prism-core.js
var prismCoreJS []byte

//go:embed assets/third_party/prism/prism-markup.js
var prismMarkupJS []byte

//go:embed assets/third_party/prism/prism-css.js
var prismCSSJS []byte

//go:embed assets/third_party/prism/prism-clike.js
var prismCLikeJS []byte

//go:embed assets/third_party/prism/prism-javascript.js
var prismJavaScriptJS []byte

//go:embed assets/third_party/prism/prism-go.js
var prismGoJS []byte

//go:embed assets/third_party/prism/prism-bash.js
var prismBashJS []byte

//go:embed assets/third_party/prism/prism-json.js
var prismJSONJS []byte

//go:embed assets/third_party/prism/prism-markdown.js
var prismMarkdownJS []byte

//go:embed assets/third_party/katex assets/third_party/beautiful-mermaid assets/mdori
var embeddedAssets embed.FS

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

func servePrismJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(prismJS)
}

func serveEmbeddedAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/_mdori/")
	if name == "." || strings.HasPrefix(name, "../") {
		http.NotFound(w, r)
		return
	}

	data, err := fs.ReadFile(embeddedAssets, "assets/"+embeddedAssetName(name))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch path.Ext(name) {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	_, _ = w.Write(data)
}

func embeddedAssetName(name string) string {
	name = strings.TrimPrefix(name, "/")
	if strings.HasPrefix(name, "vendor/") {
		return "third_party/" + strings.TrimPrefix(name, "vendor/")
	}
	return name
}
