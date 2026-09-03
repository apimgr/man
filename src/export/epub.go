package export

import (
	"fmt"
	"html"
	"os"
	"path/filepath"

	epub "github.com/bmaupin/go-epub"
)

// BuildEPUB packages the document as an EPUB 3 e-book. The bmaupin/go-epub
// library writes to disk, so we route to a temp directory and read back the
// bytes — there is no in-memory writer in v1.
func BuildEPUB(doc *Document) ([]byte, error) {
	book := epub.NewEpub(doc.Title)
	book.SetAuthor("casman")
	if doc.Subtitle != "" {
		book.SetDescription(doc.Subtitle)
	}

	for _, page := range doc.Pages {
		body := page.ContentHTML
		if body == "" {
			body = "<pre>" + html.EscapeString(page.ContentText) + "</pre>"
		}
		section := fmt.Sprintf(`<h1>%s(%s)</h1><p><em>%s</em></p>`,
			html.EscapeString(page.Name),
			html.EscapeString(page.Section),
			html.EscapeString(page.Title),
		)
		if page.Synopsis != "" {
			section += `<h2>Synopsis</h2><pre>` + html.EscapeString(page.Synopsis) + `</pre>`
		}
		section += body
		title := fmt.Sprintf("%s(%s)", page.Name, page.Section)
		if _, err := book.AddSection(section, title, "", ""); err != nil {
			return nil, fmt.Errorf("epub add section %s: %w", title, err)
		}
	}

	tmp, err := os.MkdirTemp("", "casman-epub-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	out := filepath.Join(tmp, "out.epub")
	if err := book.Write(out); err != nil {
		return nil, fmt.Errorf("epub write: %w", err)
	}
	return os.ReadFile(out)
}
