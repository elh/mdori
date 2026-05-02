# mdori

A fast, pretty, local Markdown previewer and HTML renderer.

[Demo](https://elh.github.io/mdori/) ·
[Examples](https://elh.github.io/mdori/examples.html) ·
[GitHub](https://github.com/elh/mdori)

## Usage

```plaintext
go install github.com/elh/mdori/cmd/mdori@latest
```

```plaintext
mdori path/to/file.md

usage: mdori [-addr host:port] [-no-open] [-o output.html] [-preview] <markdown-file>
  -addr string
    	listen address (default "127.0.0.1:0")
  -no-open
    	do not open the browser automatically
  -o string
    	write standalone HTML to output path and exit
  -preview
    	write a temporary standalone HTML file, open it, and exit
```

## Features

- Live browser preview with reload on save
- GitHub-flavored Markdown, footnotes, tables, task lists, and alerts
- Syntax highlighting with Prism
- Math expressions with KaTeX
- Mermaid diagrams with beautiful-mermaid
- Standalone HTML export

## How it works

By default, mdori runs a local HTTP server from the current working directory,
opens the rendered Markdown page in the browser, watches the source file, and
live reloads the page when the file changes.

This server-based mode is directory-aware: local `.md` links inside the current
working directory are served through mdori and rendered as HTML, while relative
images and assets are served from the same directory tree.

`-o` renders a single standalone HTML file and exits. `-preview` renders a
temporary standalone HTML file, opens it in the browser, and exits. These modes
do not run a server, do not live reload, and do not render linked Markdown
files. Relative `.md` links may open as raw Markdown or fail, root-relative
links do not map to mdori's serving root, and relative images/assets only work
when their paths are valid from the generated HTML file.

Standalone exports inline the assets needed by that document. Plain Markdown
stays small; math and Mermaid pages include their browser runtimes so the HTML
works offline.

### Limitations

- Mermaid support uses beautiful-mermaid's subset, not full Mermaid.
- GeoJSON/TopoJSON/STL diagrams are not supported.
- Emoji shortcodes and repository-specific autolinks are not supported.
- The live server is intended for local preview. Not hardened for untrusted inputs.
