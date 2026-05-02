# Vendored browser assets

mdori vendors only the browser runtime files it serves from the embedded
single binary.

- `vendor/prism`: Prism.js 1.30.0 components from `https://unpkg.com/prismjs@1.30.0/`.
- `vendor/katex`: KaTeX 0.16.45 runtime files from the npm package.
- `vendor/beautiful-mermaid`: a browser bundle generated from
  `beautiful-mermaid@1.1.3` with `elkjs@0.11.0` and `entities@7.0.1`.
- `mdori`: small mdori-owned glue scripts that initialize optional renderers.

Keep third-party license files next to their vendored runtime files.
