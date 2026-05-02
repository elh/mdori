# Prism assets

Prism.js files are vendored from `prismjs@1.30.0` and embedded into the mdori
binary with `go:embed`. The JavaScript files are concatenated in `internal/mdori/assets.go` and served
as `/_mdori/prism.js`.

Downloaded from `https://unpkg.com/prismjs@1.30.0/`.
