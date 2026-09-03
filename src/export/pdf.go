package export

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// BuildPDF renders a Document as a multi-section PDF. Layout is:
//   * Title page with the document title and subtitle.
//   * One chapter per page in doc.Pages, page-broken between chapters.
//   * Synopsis rendered in monospace; body text from page.ContentText so we
//     stay independent of the HTML toolchain.
func BuildPDF(doc *Document) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(true, 25)

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 24)
	pdf.MultiCell(0, 12, doc.Title, "", "C", false)
	if doc.Subtitle != "" {
		pdf.SetFont("Helvetica", "", 14)
		pdf.Ln(4)
		pdf.MultiCell(0, 8, doc.Subtitle, "", "C", false)
	}
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 6, fmt.Sprintf("%d page(s)", len(doc.Pages)), "", "C", false)

	for _, page := range doc.Pages {
		pdf.AddPage()

		pdf.SetFont("Helvetica", "B", 18)
		pdf.MultiCell(0, 10, fmt.Sprintf("%s(%s)", page.Name, page.Section), "", "L", false)

		pdf.SetFont("Helvetica", "I", 11)
		pdf.MultiCell(0, 6, page.Title, "", "L", false)

		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 5, fmt.Sprintf("Platform: %s   Section: %s", page.Platform, page.Section), "", "L", false)
		pdf.Ln(4)

		if page.Synopsis != "" {
			pdf.SetFont("Helvetica", "B", 12)
			pdf.MultiCell(0, 7, "SYNOPSIS", "", "L", false)
			pdf.SetFont("Courier", "", 9)
			pdf.MultiCell(0, 5, page.Synopsis, "", "L", false)
			pdf.Ln(3)
		}

		body := page.ContentText
		if body == "" {
			body = page.ContentMarkdown
		}
		if body == "" {
			body = "[No text content available]"
		}
		pdf.SetFont("Helvetica", "B", 12)
		pdf.MultiCell(0, 7, "DESCRIPTION", "", "L", false)
		pdf.SetFont("Courier", "", 9)
		pdf.MultiCell(0, 4.5, body, "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}
