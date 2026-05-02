package mdori

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func renderStandaloneFile(sourcePath, outputPath string) error {
	page, err := renderStandalone(sourcePath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, page, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func renderPreviewFile(sourcePath string) (string, error) {
	page, err := renderStandalone(sourcePath)
	if err != nil {
		return "", err
	}

	file, err := os.CreateTemp("", "mdori-*.html")
	if err != nil {
		return "", fmt.Errorf("create preview file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(page); err != nil {
		return "", fmt.Errorf("write preview file: %w", err)
	}
	return file.Name(), nil
}

func renderStandalone(sourcePath string) ([]byte, error) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read markdown file: %w", err)
	}

	rendered, err := newRenderer().render(source)
	if err != nil {
		return nil, fmt.Errorf("render markdown: %w", err)
	}

	page, err := renderPageWithOptions(pageTitle(sourcePath), rendered, false)
	if err != nil {
		return nil, fmt.Errorf("render page: %w", err)
	}

	out := string(page)
	out = strings.ReplaceAll(out, `<script defer src="/_mdori/prism.js"></script>`, inlineScript(prismJS))
	if rendered.NeedsMath {
		katexCSS, err := standaloneKatexCSS()
		if err != nil {
			return nil, err
		}
		katexJS, err := embeddedAsset("vendor/katex/katex.min.js")
		if err != nil {
			return nil, err
		}
		mathJS, err := embeddedAsset("mdori/math.js")
		if err != nil {
			return nil, err
		}
		out = strings.ReplaceAll(out, `<link rel="stylesheet" href="/_mdori/vendor/katex/katex.min.css">`, inlineStyle(katexCSS))
		out = strings.ReplaceAll(out, `<script defer src="/_mdori/vendor/katex/katex.min.js"></script>`, inlineScript(katexJS))
		out = strings.ReplaceAll(out, `<script defer src="/_mdori/mdori/math.js"></script>`, inlineScript(mathJS))
	}
	if rendered.NeedsMermaid {
		beautifulMermaidJS, err := embeddedAsset("vendor/beautiful-mermaid/beautiful-mermaid.min.js")
		if err != nil {
			return nil, err
		}
		mermaidJS, err := embeddedAsset("mdori/mermaid.js")
		if err != nil {
			return nil, err
		}
		out = strings.ReplaceAll(out, `<script defer src="/_mdori/vendor/beautiful-mermaid/beautiful-mermaid.min.js"></script>`, inlineScript(beautifulMermaidJS))
		out = strings.ReplaceAll(out, `<script defer src="/_mdori/mdori/mermaid.js"></script>`, inlineScript(mermaidJS))
	}

	return []byte(out), nil
}

func standaloneKatexCSS() ([]byte, error) {
	rawCSS, err := embeddedAsset("vendor/katex/katex.min.css")
	if err != nil {
		return nil, err
	}
	css := string(rawCSS)
	fonts, err := fs.ReadDir(embeddedAssets, "assets/vendor/katex/fonts")
	if err != nil {
		return nil, fmt.Errorf("read katex fonts: %w", err)
	}

	for _, font := range fonts {
		if font.IsDir() || !strings.HasSuffix(font.Name(), ".woff2") {
			continue
		}
		data, err := embeddedAsset("vendor/katex/fonts/" + font.Name())
		if err != nil {
			return nil, err
		}
		url := "data:font/woff2;base64," + base64.StdEncoding.EncodeToString(data)
		css = strings.ReplaceAll(css, "url(fonts/"+font.Name()+")", "url("+url+")")
	}
	return []byte(css), nil
}

func inlineScript(script []byte) string {
	body := strings.ReplaceAll(string(script), "</script", `<\/script`)
	return "<script>" + body + "</script>"
}

func inlineStyle(style []byte) string {
	body := strings.ReplaceAll(string(style), "</style", `<\/style`)
	return "<style>" + body + "</style>"
}

func embeddedAsset(name string) ([]byte, error) {
	data, err := fs.ReadFile(embeddedAssets, "assets/"+name)
	if err != nil {
		return nil, fmt.Errorf("read embedded asset %q: %w", name, err)
	}
	return data, nil
}

func fileURL(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
}
