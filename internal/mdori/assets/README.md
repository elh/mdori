# Prism JavaScript assets

Prism.js files are vendored from `prismjs@1.30.0` and embedded into the mdori
binary with `go:embed`. The JavaScript files are concatenated in
`internal/mdori/assets.go` and served as `/_mdori/prism.js`.

mdori owns Prism styling in the page template so syntax colors can share the
same light/dark theme tokens as the rest of the renderer.

Downloaded from `https://unpkg.com/prismjs@1.30.0/`.
