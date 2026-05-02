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
  <link rel="stylesheet" href="/_mdori/prism.css">
  <script defer src="/_mdori/prism.js"></script>
  <style>
    :root {
      color-scheme: light;
      --bg: #ffffff;
      --text: #1f2328;
      --muted: #59636e;
      --border: #d0d7de;
      --code-bg: #f6f8fa;
      --quote: #8c959f;
      --link: #0f5c83;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", Helvetica, Arial, sans-serif;
      line-height: 1.7;
    }

    main {
      width: min(100%, 52rem);
      margin: 0 auto;
      padding: 3rem 1.5rem 4rem;
    }

    article {
      padding: 2.5rem;
    }

    h1, h2, h3, h4, h5, h6 {
      line-height: 1.2;
      margin: 2rem 0 0.75rem;
    }

    h1 {
      margin-top: 0;
      font-size: clamp(2.2rem, 5vw, 3rem);
    }

    p, ul, ol, pre, table, blockquote {
      margin: 1rem 0;
    }

    a {
      color: var(--link);
    }

    code, pre {
      font-family: ui-monospace, "SFMono-Regular", "SF Mono", Menlo, Consolas, monospace;
    }

    code {
      background: var(--code-bg);
      border-radius: 0.35rem;
      padding: 0.1rem 0.35rem;
      font-size: 0.92em;
    }

    pre {
      background: var(--code-bg);
      border-radius: 14px;
      padding: 1rem 1.1rem;
      overflow-x: auto;
    }

    pre code {
      background: transparent;
      padding: 0;
    }

    blockquote {
      border-left: 4px solid var(--quote);
      color: var(--muted);
      padding-left: 1rem;
    }

    table {
      width: 100%;
      border-collapse: collapse;
    }

    th, td {
      border: 1px solid var(--border);
      padding: 0.6rem 0.75rem;
      text-align: left;
      vertical-align: top;
    }

    img {
      max-width: 100%;
      height: auto;
      border-radius: 10px;
    }

    hr {
      border: 0;
      border-top: 1px solid var(--border);
      margin: 2rem 0;
    }

    @media (max-width: 640px) {
      main {
        padding: 1rem 0.75rem 2rem;
      }

      article {
        padding: 1.25rem;
      }
    }
  </style>
</head>
<body>
  <main>
    <article>
      {{ .Body }}
    </article>
  </main>
  <script>
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
}

func renderPage(title string, body template.HTML) ([]byte, error) {
	var buf bytes.Buffer
	err := pageTmpl.Execute(&buf, pageData{
		Title: title,
		Body:  body,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
