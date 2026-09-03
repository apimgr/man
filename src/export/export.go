// Package export builds offline-readable bundles of man pages in PDF, EPUB,
// and standalone HTML formats. See AI.md PART 16 and IDEA.md "Export
// (PDF/EPUB)".
//
// Each builder takes a Document (one or many pages) and returns the encoded
// bytes plus the appropriate MIME type. The handlers in src/server/handler/
// stream those bytes to the response.
package export

import (
	"errors"
	"fmt"
	"strings"

	"github.com/casapps/casman/src/server/model"
)

// Format names the supported output bundles.
type Format string

const (
	FormatPDF  Format = "pdf"
	FormatEPUB Format = "epub"
	FormatHTML Format = "html"
)

// Available returns the formats the export package can produce. The order is
// stable so the admin UI and /api/v1/export/formats stay deterministic.
func Available() []Format {
	return []Format{FormatPDF, FormatEPUB, FormatHTML}
}

// FormatInfo describes a format for the API response.
type FormatInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Extension   string `json:"extension"`
	ContentType string `json:"content_type"`
	Description string `json:"description"`
}

// Catalog returns the JSON-friendly list of formats.
func Catalog() []FormatInfo {
	return []FormatInfo{
		{ID: "pdf", Name: "PDF", Extension: ".pdf", ContentType: "application/pdf", Description: "Print-quality PDF; one chapter per man page."},
		{ID: "epub", Name: "EPUB", Extension: ".epub", ContentType: "application/epub+zip", Description: "EPUB 3 e-book; readable on Kindle, Kobo, iBooks, and other readers."},
		{ID: "html", Name: "HTML", Extension: ".html", ContentType: "text/html; charset=utf-8", Description: "Standalone HTML with embedded styles for offline viewing."},
	}
}

// ContentType returns the response Content-Type for the given format.
func (f Format) ContentType() string {
	for _, fi := range Catalog() {
		if fi.ID == string(f) {
			return fi.ContentType
		}
	}
	return "application/octet-stream"
}

// Extension returns the file extension (with leading dot) for the format.
func (f Format) Extension() string {
	for _, fi := range Catalog() {
		if fi.ID == string(f) {
			return fi.Extension
		}
	}
	return ""
}

// ParseFormat normalizes user input (with or without leading dot) into a
// Format. Unknown values return an error so handlers can 404 cleanly.
func ParseFormat(s string) (Format, error) {
	s = strings.ToLower(strings.TrimPrefix(s, "."))
	for _, fi := range Catalog() {
		if fi.ID == s {
			return Format(fi.ID), nil
		}
	}
	return "", fmt.Errorf("export: unsupported format %q", s)
}

// Document is the input to every builder. A single-page export sets Pages to
// a one-element slice.
type Document struct {
	Title    string
	Subtitle string
	Pages    []*model.ManPage
}

// Build dispatches to the right format-specific builder and returns the bytes
// + content type.
func Build(doc *Document, format Format) ([]byte, string, error) {
	if doc == nil || len(doc.Pages) == 0 {
		return nil, "", errors.New("export: empty document")
	}
	switch format {
	case FormatPDF:
		b, err := BuildPDF(doc)
		return b, format.ContentType(), err
	case FormatEPUB:
		b, err := BuildEPUB(doc)
		return b, format.ContentType(), err
	case FormatHTML:
		b, err := BuildHTML(doc)
		return b, format.ContentType(), err
	default:
		return nil, "", fmt.Errorf("export: unknown format %q", format)
	}
}
