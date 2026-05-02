package mdori

import (
	"bytes"
	"html/template"
)

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <script>
    (() => {
      const themes = new Set(["system", "light", "dark"]);
      let theme = "system";
      try {
        const stored = localStorage.getItem("mdori-theme");
        if (themes.has(stored)) {
          theme = stored;
        }
      } catch {
      }
      document.documentElement.dataset.theme = theme;
    })();
  </script>
  <link rel="stylesheet" href="/_mdori/prism.css">
  <script defer src="/_mdori/prism.js"></script>
  <style>
    :root {
      color-scheme: light;
      --color-bg: #ffffff;
      --color-text: #1f2328;
      --color-muted: #59636e;
      --color-border: #d0d7de;
      --color-code-bg: #f6f8fa;
      --color-control-bg: #ffffff;
      --color-control-text: #1f2328;
      --color-quote: #8c959f;
      --color-link: #0f5c83;
      --color-kbd-shadow: #d0d7de;
      --color-alert-note: #0969da;
      --color-alert-tip: #1a7f37;
      --color-alert-important: #8250df;
      --color-alert-warning: #9a6700;
      --color-alert-caution: #cf222e;
      --color-syntax-text: #24292f;
      --color-syntax-comment: #6e7781;
      --color-syntax-punctuation: #57606a;
      --color-syntax-keyword: #cf222e;
      --color-syntax-function: #8250df;
      --color-syntax-string: #116329;
      --color-syntax-number: #0550ae;
      --color-syntax-operator: #0550ae;
      --color-syntax-selection: #b3d4fc;
      --font-body: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", Helvetica, Arial, sans-serif;
      --font-mono: ui-monospace, "SFMono-Regular", "SF Mono", Menlo, Consolas, monospace;
      --text-sm: 0.875rem;
      --text-code: 0.92em;
      --text-h1: 2rem;
      --weight-normal: 400;
      --weight-semibold: 600;
      --weight-bold: 700;
      --leading-body: 1.6;
      --leading-heading: 1.2;
      --leading-code: 1.5;
      --leading-small: 1.4;
      --space-block: 1rem;
      --space-page-x: 1.5rem;
      --space-page-y: 3rem;
      --space-heading: 2rem 0 0.75rem;
      --space-inline-code: 0.1rem 0.35rem;
      --space-code-block: 1rem 1.1rem;
      --space-table-cell: 0.6rem 0.75rem;
      --radius-code: 0.35rem;
      --radius-code-block: 6px;
      --width-content: 64rem;
      --width-toc: 12rem;
    }

    html[data-theme="dark"] {
      color-scheme: dark;
      --color-bg: #0d1117;
      --color-text: #e6edf3;
      --color-muted: #8b949e;
      --color-border: #30363d;
      --color-code-bg: #161b22;
      --color-control-bg: #161b22;
      --color-control-text: #e6edf3;
      --color-quote: #6e7681;
      --color-link: #79c0ff;
      --color-kbd-shadow: #30363d;
      --color-alert-note: #2f81f7;
      --color-alert-tip: #3fb950;
      --color-alert-important: #a371f7;
      --color-alert-warning: #d29922;
      --color-alert-caution: #f85149;
      --color-syntax-text: #e6edf3;
      --color-syntax-comment: #8b949e;
      --color-syntax-punctuation: #c9d1d9;
      --color-syntax-keyword: #ff7b72;
      --color-syntax-function: #d2a8ff;
      --color-syntax-string: #a5d6ff;
      --color-syntax-number: #79c0ff;
      --color-syntax-operator: #ff7b72;
      --color-syntax-selection: #264f78;
    }

    @media (prefers-color-scheme: dark) {
      html[data-theme="system"] {
        color-scheme: dark;
        --color-bg: #0d1117;
        --color-text: #e6edf3;
        --color-muted: #8b949e;
        --color-border: #30363d;
        --color-code-bg: #161b22;
        --color-control-bg: #161b22;
        --color-control-text: #e6edf3;
        --color-quote: #6e7681;
        --color-link: #79c0ff;
        --color-kbd-shadow: #30363d;
        --color-alert-note: #2f81f7;
        --color-alert-tip: #3fb950;
        --color-alert-important: #a371f7;
        --color-alert-warning: #d29922;
        --color-alert-caution: #f85149;
        --color-syntax-text: #e6edf3;
        --color-syntax-comment: #8b949e;
        --color-syntax-punctuation: #c9d1d9;
        --color-syntax-keyword: #ff7b72;
        --color-syntax-function: #d2a8ff;
        --color-syntax-string: #a5d6ff;
        --color-syntax-number: #79c0ff;
        --color-syntax-operator: #ff7b72;
        --color-syntax-selection: #264f78;
      }
    }

    * {
      box-sizing: border-box;
    }

    html {
      scroll-padding-top: var(--space-block);
    }

    body {
      margin: 0;
      background: var(--color-bg);
      color: var(--color-text);
      font-family: var(--font-body);
      font-weight: var(--weight-normal);
      line-height: var(--leading-body);
    }

    .theme-select {
      position: fixed;
      top: var(--space-block);
      right: var(--space-block);
      z-index: 1;
      max-width: calc(100vw - 2rem);
      background: var(--color-control-bg);
      color: var(--color-control-text);
      border: 1px solid var(--color-border);
      border-radius: 0.375rem;
      font: inherit;
      font-size: var(--text-sm);
      line-height: var(--leading-small);
      padding: 0.25rem 0.5rem;
    }

    .toc {
      display: none;
    }

    main {
      width: min(100%, var(--width-content));
      margin: 0 auto;
      padding: var(--space-page-y) var(--space-page-x) 4rem;
    }

    article {
      padding: 0;
    }

    h1, h2, h3, h4, h5, h6 {
      line-height: var(--leading-heading);
      margin: var(--space-heading);
    }

    h1 {
      margin-top: 0;
      font-size: var(--text-h1);
    }

    p, ul, ol, pre, blockquote, .table-scroll {
      margin: var(--space-block) 0;
    }

    a {
      color: var(--color-link);
    }

    :target {
      scroll-margin-top: var(--space-block);
    }

    code, pre {
      font-family: var(--font-mono);
    }

    code {
      background: var(--color-code-bg);
      border-radius: var(--radius-code);
      padding: var(--space-inline-code);
      font-size: var(--text-code);
    }

    pre {
      background: var(--color-code-bg);
      border-radius: var(--radius-code-block);
      padding: var(--space-code-block);
      overflow-x: auto;
    }

    pre code {
      background: transparent;
      padding: 0;
    }

    blockquote {
      border-left: 4px solid var(--color-quote);
      color: var(--color-muted);
      padding-left: var(--space-block);
    }

    .markdown-alert {
      border-left: 4px solid var(--color-alert-note);
      margin: var(--space-block) 0;
      padding: 0.1rem var(--space-block);
    }

    .markdown-alert p {
      margin: 0.75rem 0;
    }

    .markdown-alert-title {
      color: var(--color-alert-note);
      font-weight: var(--weight-semibold);
    }

    .markdown-alert-tip {
      border-left-color: var(--color-alert-tip);
    }

    .markdown-alert-tip .markdown-alert-title {
      color: var(--color-alert-tip);
    }

    .markdown-alert-important {
      border-left-color: var(--color-alert-important);
    }

    .markdown-alert-important .markdown-alert-title {
      color: var(--color-alert-important);
    }

    .markdown-alert-warning {
      border-left-color: var(--color-alert-warning);
    }

    .markdown-alert-warning .markdown-alert-title {
      color: var(--color-alert-warning);
    }

    .markdown-alert-caution {
      border-left-color: var(--color-alert-caution);
    }

    .markdown-alert-caution .markdown-alert-title {
      color: var(--color-alert-caution);
    }

    kbd {
      background: var(--color-code-bg);
      border: 1px solid var(--color-border);
      border-bottom-width: 2px;
      border-radius: 0.25rem;
      box-shadow: inset 0 -1px 0 var(--color-kbd-shadow);
      font-family: var(--font-mono);
      font-size: 0.85em;
      padding: var(--space-inline-code);
    }

    .table-scroll {
      overflow-x: auto;
      width: 100%;
    }

    table {
      border-collapse: collapse;
      margin: 0;
      min-width: 100%;
    }

    th, td {
      border: 1px solid var(--color-border);
      min-width: 10rem;
      max-width: 28rem;
      overflow-wrap: anywhere;
      padding: var(--space-table-cell);
      text-align: left;
      vertical-align: top;
    }

    img {
      max-width: 100%;
      height: auto;
    }

    hr {
      border: 0;
      border-top: 1px solid var(--color-border);
      margin: 2rem 0;
    }

    code[class*="language-"], pre[class*="language-"] {
      color: var(--color-syntax-text);
      background: transparent;
      text-shadow: none;
      font-family: var(--font-mono);
      line-height: var(--leading-code);
    }

    pre[class*="language-"] {
      background: var(--color-code-bg);
      margin: var(--space-block) 0;
      padding: var(--space-code-block);
    }

    :not(pre) > code[class*="language-"] {
      background: var(--color-code-bg);
    }

    code[class*="language-"] ::selection,
    code[class*="language-"]::selection,
    pre[class*="language-"] ::selection,
    pre[class*="language-"]::selection {
      background: var(--color-syntax-selection);
      text-shadow: none;
    }

    .token.comment,
    .token.prolog,
    .token.doctype,
    .token.cdata {
      color: var(--color-syntax-comment);
    }

    .token.punctuation {
      color: var(--color-syntax-punctuation);
    }

    .token.keyword,
    .token.atrule,
    .token.attr-value {
      color: var(--color-syntax-keyword);
    }

    .token.function,
    .token.class-name {
      color: var(--color-syntax-function);
    }

    .token.string,
    .token.attr-name,
    .token.char,
    .token.builtin,
    .token.inserted,
    .token.selector {
      color: var(--color-syntax-string);
    }

    .token.number,
    .token.boolean,
    .token.constant,
    .token.property,
    .token.symbol,
    .token.tag,
    .token.deleted {
      color: var(--color-syntax-number);
    }

    .token.operator,
    .token.entity,
    .token.url,
    .language-css .token.string,
    .style .token.string {
      color: var(--color-syntax-operator);
      background: transparent;
    }

    .token.bold,
    .token.important {
      font-weight: var(--weight-bold);
    }

    @media (min-width: 92rem) {
      .toc {
        display: block;
        position: fixed;
        top: var(--space-page-y);
        left: max(var(--space-page-x), calc((100vw - var(--width-content)) / 2 - 14rem));
        width: var(--width-toc);
        max-height: calc(100vh - 6rem);
        overflow-y: auto;
        padding-right: var(--space-block);
        font-size: var(--text-sm);
        line-height: var(--leading-small);
      }

      .toc-link {
        display: block;
        margin: 0.35rem 0;
        color: var(--color-muted);
        text-decoration: none;
      }

      .toc-link:hover {
        color: var(--color-link);
        text-decoration: underline;
      }

      .toc-level-1 {
        font-weight: var(--weight-semibold);
      }

      .toc-level-3 {
        padding-left: var(--space-block);
      }
    }

    @media (max-width: 64rem) {
      .theme-select {
        display: none;
      }
    }

    @media (max-width: 640px) {
      main {
        padding: var(--space-block) 0.75rem 2rem;
      }

      article {
        padding: 1.25rem;
      }
    }
  </style>
</head>
<body>
  <select class="theme-select" aria-label="Color theme">
    <option value="system">System</option>
    <option value="light">Light</option>
    <option value="dark">Dark</option>
  </select>
  {{ if .TOC }}
  <nav class="toc" aria-label="Table of contents">
    {{ range .TOC }}
    <a class="toc-link toc-level-{{ .Level }}" href="#{{ .ID }}">{{ .Text }}</a>
    {{ end }}
  </nav>
  {{ end }}
  <main>
    <article>
      {{ .Body }}
    </article>
  </main>
  <script>
    (() => {
      const themes = new Set(["system", "light", "dark"]);
      const normalizeTheme = (theme) => themes.has(theme) ? theme : "system";

      document.addEventListener("DOMContentLoaded", () => {
        const select = document.querySelector(".theme-select");
        if (!select) {
          return;
        }

        select.value = normalizeTheme(document.documentElement.dataset.theme);
        select.addEventListener("change", () => {
          const theme = normalizeTheme(select.value);
          document.documentElement.dataset.theme = theme;
          try {
            localStorage.setItem("mdori-theme", theme);
          } catch {
          }
        });
      });
    })();

    const events = new EventSource("/events");
    events.addEventListener("reload", () => {
      window.location.reload();
    });
  </script>
</body>
</html>
`))

type pageData struct {
	Title string
	Body  template.HTML
	TOC   []tocItem
}

func renderPage(title string, body template.HTML, toc []tocItem) ([]byte, error) {
	var buf bytes.Buffer
	err := pageTmpl.Execute(&buf, pageData{
		Title: title,
		Body:  body,
		TOC:   toc,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
