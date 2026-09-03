// Export handlers: per-page download (.pdf/.epub/.html alongside the existing
// .txt/.md/.raw extensions) plus bulk export by section or platform. See
// AI.md PART 16 and IDEA.md "Export (PDF/EPUB)".

package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/casapps/casman/src/export"
	"github.com/casapps/casman/src/server/model"
)

// APIExportFormats handles GET /api/v1/export/formats and lists the bundles
// the export package can produce.
func (h *Handlers) APIExportFormats(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, map[string]interface{}{
		"formats": export.Catalog(),
	}, http.StatusOK)
}

// servePageExport writes a single-page export bundle to the response. The
// caller is expected to have already validated the format. The downloaded
// file name follows {name}.{section}.{ext} so multiple downloads do not
// collide.
func (h *Handlers) servePageExport(w http.ResponseWriter, page *model.ManPage, format export.Format) {
	doc := &export.Document{
		Title:    fmt.Sprintf("%s(%s)", page.Name, page.Section),
		Subtitle: page.Title,
		Pages:    []*model.ManPage{page},
	}
	body, contentType, err := export.Build(doc, format)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, fmt.Sprintf("export: %v", err))
		return
	}
	filename := fmt.Sprintf("%s.%s%s", page.Name, page.Section, format.Extension())
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// ExportSection handles /export/section/{section}.{ext}, bundling every page
// in that section into a single document.
func (h *Handlers) ExportSection(w http.ResponseWriter, r *http.Request) {
	rawSection := chi.URLParam(r, "section")
	sectionID, ext := splitExt(rawSection)
	format, err := export.ParseFormat(ext)
	if err != nil {
		h.renderError(w, http.StatusNotFound, err.Error())
		return
	}
	if h.db == nil {
		h.renderError(w, http.StatusServiceUnavailable, "Database not initialized")
		return
	}
	pages, err := h.collectBulkPages("", sectionID)
	if err != nil || len(pages) == 0 {
		h.renderError(w, http.StatusNotFound, fmt.Sprintf("No pages found in section %s", sectionID))
		return
	}
	h.serveBulkExport(w, pages, format,
		fmt.Sprintf("section-%s%s", sectionID, format.Extension()),
		fmt.Sprintf("Section %s", sectionID),
		fmt.Sprintf("%d page(s) from section %s", len(pages), sectionID),
	)
}

// ExportPlatform handles /export/platform/{platform}.{ext}, bundling every
// page for that platform.
func (h *Handlers) ExportPlatform(w http.ResponseWriter, r *http.Request) {
	rawPlatform := chi.URLParam(r, "platform")
	platformID, ext := splitExt(rawPlatform)
	format, err := export.ParseFormat(ext)
	if err != nil {
		h.renderError(w, http.StatusNotFound, err.Error())
		return
	}
	if h.db == nil {
		h.renderError(w, http.StatusServiceUnavailable, "Database not initialized")
		return
	}
	pages, err := h.collectBulkPages(platformID, "")
	if err != nil || len(pages) == 0 {
		h.renderError(w, http.StatusNotFound, fmt.Sprintf("No pages found for platform %s", platformID))
		return
	}
	h.serveBulkExport(w, pages, format,
		fmt.Sprintf("platform-%s%s", platformID, format.Extension()),
		fmt.Sprintf("Platform: %s", platformID),
		fmt.Sprintf("%d page(s) for %s", len(pages), platformID),
	)
}

func (h *Handlers) serveBulkExport(w http.ResponseWriter, pages []*model.ManPage, format export.Format, filename, title, subtitle string) {
	doc := &export.Document{Title: title, Subtitle: subtitle, Pages: pages}
	body, contentType, err := export.Build(doc, format)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, fmt.Sprintf("export: %v", err))
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// collectBulkPages walks the Browse() listing and resolves each summary into
// a full ManPage. Bulk export is rare and not on the hot path, so the simple
// page-by-page fetch is adequate.
func (h *Handlers) collectBulkPages(platform, section string) ([]*model.ManPage, error) {
	const pageSize = 200
	const safetyCap = 5000

	var out []*model.ManPage
	for page := 1; ; page++ {
		summaries, _, err := h.db.Browse(section, platform, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(summaries) == 0 {
			break
		}
		for _, s := range summaries {
			full, err := h.db.GetManPage(s.Platform, s.Section, s.Name)
			if err == nil && full != nil {
				out = append(out, full)
				if len(out) >= safetyCap {
					return out, nil
				}
			}
		}
		if len(summaries) < pageSize {
			break
		}
	}
	return out, nil
}

// splitExt separates `name.ext` into ("name", "ext"). Returns the original
// string and "" when there is no dot.
func splitExt(s string) (string, string) {
	if i := strings.LastIndexByte(s, '.'); i > 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
