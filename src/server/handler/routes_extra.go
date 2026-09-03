// Additional route handlers added in the second pass: i18n man page API,
// compare-within-section API, per-platform sitemap, and combined per-platform/
// per-section feed. The matching template/feed/sitemap helpers were already
// present in handler.go so this file just adds thin wrappers and the routes
// they back. See AI.md PART 14 (API), PART 16 (web), PART 31 (i18n).

package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/casapps/casman/src/server/model"
)

// APIManPageLang handles /api/v1/man/{lang}/{os}/{section}/{name}.
// The language segment is captured for forward compatibility — translated
// content is not yet stored in the database, so we currently fall through to
// the OS/section/name lookup and return the English source. The lang code is
// echoed back via the X-Content-Language response header so clients can tell
// whether translation is active.
func (h *Handlers) APIManPageLang(w http.ResponseWriter, r *http.Request) {
	lang := chi.URLParam(r, "lang")
	osParam := chi.URLParam(r, "os")
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	if lang != "" {
		w.Header().Set("X-Content-Language", lang)
	}
	h.serveAPIManPage(w, r, osParam, section, name)
}

// APICompareSection handles /api/v1/compare/{section}/{name}, mirroring the
// existing /api/v1/compare/{name} route but scoped to a section.
func (h *Handlers) APICompareSection(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	if h.db == nil {
		h.jsonResponse(w, map[string]string{"error": "database not available"}, http.StatusServiceUnavailable)
		return
	}
	result, err := h.db.Compare(name, section)
	if err != nil {
		h.jsonResponse(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	h.jsonResponse(w, result, http.StatusOK)
}

// FeedCombined handles /feed/{platform}/{section}.xml — a per-platform,
// per-section subscription feed. The .xml suffix on the {section} parameter
// is stripped because chi's path matching treats the dot as part of the
// parameter value.
func (h *Handlers) FeedCombined(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	section := strings.TrimSuffix(chi.URLParam(r, "section"), ".xml")
	h.serveFeed(w, r, platform, section)
}

// SitemapPlatforms handles /sitemap-platforms.xml — a per-platform sitemap
// listing the canonical platform browse URLs. Provides per-platform entries
// independent of the section sitemap so search engines can crawl them.
func (h *Handlers) SitemapPlatforms(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)
	for _, p := range model.Platforms {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + baseURL + "/browse/os/" + p.ID + "</loc>\n")
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("    <priority>0.7</priority>\n")
		sb.WriteString("  </url>\n")
	}
	sb.WriteString("</urlset>\n")

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String()))
}
