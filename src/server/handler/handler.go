// Package handler implements HTTP request handlers for casman.
package handler

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/casapps/casman/src/config"
	"github.com/casapps/casman/src/server/data"
	"github.com/casapps/casman/src/server/model"
	"github.com/casapps/casman/src/server/static"
	"github.com/casapps/casman/src/server/store"
	"github.com/casapps/casman/src/server/template"
)

// Handlers holds the HTTP handlers and their dependencies.
type Handlers struct {
	cfg       *config.Config
	version   string
	commitID  string
	buildDate string
	startTime time.Time
	db        *store.DB
	tmpl      *template.Templates
}

// New creates a new Handlers instance.
func New(cfg *config.Config, version, commitID, buildDate string) *Handlers {
	h := &Handlers{
		cfg:       cfg,
		version:   version,
		commitID:  commitID,
		buildDate: buildDate,
		startTime: time.Now(),
	}

	return h
}

// Init initializes the database and templates.
// If dbPath is empty or file doesn't exist, uses embedded database.
func (h *Handlers) Init(dbPath string) error {
	var err error

	// Try to use external database first
	if dbPath != "" {
		if _, statErr := os.Stat(dbPath); statErr == nil {
			h.db, err = store.New(dbPath)
			if err == nil {
				goto loadTemplates
			}
		}
	}

	// Fall back to embedded database
	err = h.initWithEmbedded()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

loadTemplates:
	h.tmpl, err = template.New()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	return nil
}

// initWithEmbedded extracts and uses the embedded database.
func (h *Handlers) initWithEmbedded() error {
	// Check if we have embedded data
	if len(data.ManPagesDB) == 0 {
		return fmt.Errorf("no embedded database available")
	}

	// Create a temporary file for the embedded database
	// Use cache directory if available, otherwise temp
	cacheDir := os.TempDir()
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		cacheDir = xdgCache
	} else if home, err := os.UserHomeDir(); err == nil {
		cacheDir = filepath.Join(home, ".cache")
	}

	dbDir := filepath.Join(cacheDir, "casman")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "manpages.db")

	// Write embedded database to file
	if err := os.WriteFile(dbPath, data.ManPagesDB, 0644); err != nil {
		return fmt.Errorf("writing embedded database: %w", err)
	}

	// Open the database
	var err error
	h.db, err = store.New(dbPath)
	if err != nil {
		return fmt.Errorf("opening embedded database: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (h *Handlers) Close() error {
	if h.db != nil {
		return h.db.Close()
	}
	return nil
}

// Response helpers

func (h *Handlers) jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func (h *Handlers) textResponse(w http.ResponseWriter, text string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(text))
	w.Write([]byte("\n"))
}

func (h *Handlers) renderTemplate(w http.ResponseWriter, name string, data interface{}, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.tmpl.Render(w, name, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handlers) renderError(w http.ResponseWriter, code int, message string) {
	data := template.ErrorData{
		Code:    code,
		Message: message,
	}
	h.renderTemplate(w, "error.html", data, code)
}

// Health endpoints

// buildHealthResponse builds the health response per PART 13.
func (h *Handlers) buildHealthResponse() model.HealthResponse {
	// Get stats from database
	var stats model.Stats
	if h.db != nil {
		stats, _ = h.db.GetStats()
	}

	// Check components
	dbStatus := "ok"
	if h.db == nil {
		dbStatus = "error"
	} else if _, err := h.db.GetStats(); err != nil {
		dbStatus = "error"
	}

	// Calculate overall status
	status := "healthy"
	if dbStatus != "ok" {
		status = "degraded"
	}

	return model.HealthResponse{
		Project: model.ProjectInfo{
			Name:        "casman",
			Tagline:     "Universal Man Pages",
			Description: "Man pages from BSD, macOS, Linux, and other Unix-like systems",
		},
		Status:    status,
		Version:   h.version,
		GoVersion: runtime.Version(),
		Build: model.BuildInfo{
			Commit: h.commitID,
			Date:   h.buildDate,
		},
		Uptime:    formatUptime(h.startTime),
		Mode:      h.cfg.Server.Mode,
		Timestamp: time.Now().UTC(),
		Cluster: model.ClusterInfo{
			Enabled: false,
		},
		Features: model.FeaturesInfo{
			Tor:   torInfoFor(),
			GeoIP: false,
		},
		Checks: model.ChecksInfo{
			Database:  dbStatus,
			Cache:     "ok",
			Disk:      "ok",
			Scheduler: "ok",
		},
		Stats: model.HealthStats{
			RequestsTotal: 0,
			Requests24h:   0,
			ActiveConns:   0,
			ManPagesTotal: stats.TotalPages,
			PlatformCount: stats.TotalPlatforms,
			SectionCount:  stats.TotalSections,
		},
		AppData: model.AppDataInfo{
			ManPages:   stats.TotalPages,
			Platforms:  stats.TotalPlatforms,
			Sections:   stats.TotalSections,
			SearchHits: 0,
		},
	}
}

// formatUptime returns a human-readable uptime string.
func formatUptime(start time.Time) string {
	d := time.Since(start)

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// Healthz handles the web health check endpoint.
func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")

	// Content negotiation per PART 14
	if strings.Contains(accept, "application/json") {
		h.APIHealthz(w, r)
		return
	}

	if strings.Contains(accept, "text/plain") {
		health := h.buildHealthResponse()
		h.textResponse(w, fmt.Sprintf("%s %s - %s", health.Project.Name, health.Version, health.Status), http.StatusOK)
		return
	}

	// HTML response
	health := h.buildHealthResponse()
	data := template.HealthData{
		Title:  "Health Status - casman",
		Health: health,
	}
	h.renderTemplate(w, "healthz.html", data, http.StatusOK)
}

// APIHealthz handles the API health check endpoint.
func (h *Handlers) APIHealthz(w http.ResponseWriter, r *http.Request) {
	health := h.buildHealthResponse()
	h.jsonResponse(w, health, http.StatusOK)
}

// Homepage

// Home handles the homepage.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	var stats model.Stats
	var popular []model.ManPageSummary
	var err error

	if h.db != nil {
		stats, err = h.db.GetStats()
		if err != nil {
			stats = model.Stats{}
		}
		popular, err = h.db.GetPopular(10)
		if err != nil {
			popular = nil
		}
	}

	data := template.HomeData{
		Title:       "casman - Universal Man Pages",
		Stats:       stats,
		Popular:     popular,
		Sections:    model.Sections,
		Platforms:   model.Platforms,
		RecentPages: nil,
	}

	h.renderTemplate(w, "home.html", data, http.StatusOK)
}

// Man page handlers

// ManPage handles /man/{name}
func (h *Handlers) ManPage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.serveManPage(w, r, "", "", name, "html")
}

// ManPageWithFormat handles /man/{name}.{ext}
func (h *Handlers) ManPageWithFormat(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	ext := chi.URLParam(r, "ext")
	h.serveManPage(w, r, "", "", name, ext)
}

// ManPageSection handles /man/{section}/{name}
func (h *Handlers) ManPageSection(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	h.serveManPage(w, r, "", section, name, "html")
}

// ManPageSectionWithFormat handles /man/{section}/{name}.{ext}
func (h *Handlers) ManPageSectionWithFormat(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	ext := chi.URLParam(r, "ext")
	h.serveManPage(w, r, "", section, name, ext)
}

// ManPageOSSection handles /man/{os}/{section}/{name}
func (h *Handlers) ManPageOSSection(w http.ResponseWriter, r *http.Request) {
	osParam := chi.URLParam(r, "os")
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	h.serveManPage(w, r, osParam, section, name, "html")
}

// ManPageOSSectionWithFormat handles /man/{os}/{section}/{name}.{ext}
func (h *Handlers) ManPageOSSectionWithFormat(w http.ResponseWriter, r *http.Request) {
	osParam := chi.URLParam(r, "os")
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	ext := chi.URLParam(r, "ext")
	h.serveManPage(w, r, osParam, section, name, ext)
}

func (h *Handlers) serveManPage(w http.ResponseWriter, r *http.Request, platform, section, name, format string) {
	if h.db == nil {
		h.renderError(w, http.StatusServiceUnavailable, "Database not initialized")
		return
	}

	var page *model.ManPage
	var err error

	if platform != "" && section != "" {
		page, err = h.db.GetManPage(platform, section, name)
	} else if section != "" {
		page, err = h.db.GetManPageByName(name, section, "")
	} else {
		page, err = h.db.GetManPageByName(name, "", "")
	}

	if err != nil || page == nil {
		h.renderError(w, http.StatusNotFound, fmt.Sprintf("Man page %s not found", name))
		return
	}

	switch format {
	case "txt", "text":
		h.textResponse(w, page.ContentText, http.StatusOK)
	case "md", "markdown":
		h.textResponse(w, page.ContentMarkdown, http.StatusOK)
	case "raw", "src", "source":
		h.textResponse(w, page.ContentRaw, http.StatusOK)
	case "json":
		h.jsonResponse(w, page, http.StatusOK)
	case "pdf":
		h.servePageExport(w, page, "pdf")
	case "epub":
		h.servePageExport(w, page, "epub")
	case "shtml", "standalone":
		h.servePageExport(w, page, "html")
	default:
		// HTML view
		otherPlatforms, _ := h.db.GetOtherPlatforms(name, section, platform)

		tldr, _ := h.db.GetTLDR(page.Name, page.Section)

		data := template.ManPageData{
			Title:          fmt.Sprintf("%s(%s) - casman", page.Name, page.Section),
			Name:           page.Name,
			Section:        page.Section,
			Platform:       page.Platform,
			PageTitle:      page.Title,
			Synopsis:       page.Synopsis,
			ContentHTML:    htmltemplate.HTML(page.ContentHTML),
			OtherPlatforms: otherPlatforms,
			SeeAlso:        page.SeeAlso,
			Bookmarked:     false,
			TLDR:           tldr,
		}
		h.renderTemplate(w, "manpage.html", data, http.StatusOK)
	}
}

// Search handlers

// Search handles /search
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	rawQuery := r.URL.Query().Get("q")
	urlSection := r.URL.Query().Get("section")
	urlPlatform := r.URL.Query().Get("platform")
	pageStr := r.URL.Query().Get("page")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	parsed := ParseSearchQuery(rawQuery).MergeURLParams(urlSection, urlPlatform)
	query, section, platform := parsed.Query, parsed.Section, parsed.Platform

	var results []model.SearchResult
	var total int
	var err error

	if h.db != nil && query != "" {
		results, total, err = h.db.Search(query, section, platform, page, 20)
		if err != nil {
			results = nil
			total = 0
		}
	}

	data := template.SearchData{
		Title:     "Search - casman",
		Query:     rawQuery,
		Section:   section,
		Platform:  platform,
		Results:   results,
		Total:     total,
		Page:      page,
		HasMore:   total > page*20,
		Sections:  model.Sections,
		Platforms: model.Platforms,
	}

	h.renderTemplate(w, "search.html", data, http.StatusOK)
}

// Browse handlers

// Browse handles /browse
func (h *Handlers) Browse(w http.ResponseWriter, r *http.Request) {
	h.serveBrowse(w, r, "", "")
}

// BrowseSection handles /browse/{section}
func (h *Handlers) BrowseSection(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	h.serveBrowse(w, r, section, "")
}

// BrowseOS handles /browse/os/{os}
func (h *Handlers) BrowseOS(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "os")
	h.serveBrowse(w, r, "", platform)
}

func (h *Handlers) serveBrowse(w http.ResponseWriter, r *http.Request, section, platform string) {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	var pages []model.ManPageSummary
	var total int
	var err error

	if h.db != nil {
		pages, total, err = h.db.Browse(section, platform, page, 50)
		if err != nil {
			pages = nil
			total = 0
		}
	}

	// Get section/platform counts
	sections := model.Sections
	platforms := model.Platforms

	if h.db != nil {
		sections, _ = h.db.GetSections()
		platforms, _ = h.db.GetPlatforms()
	}

	data := template.BrowseData{
		Title:     "Browse - casman",
		Section:   section,
		Platform:  platform,
		Pages:     pages,
		Total:     total,
		Page:      page,
		HasMore:   total > page*50,
		Sections:  sections,
		Platforms: platforms,
	}

	h.renderTemplate(w, "browse.html", data, http.StatusOK)
}

// Compare handlers

// Compare handles /compare/{name}
func (h *Handlers) Compare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.serveCompare(w, r, "", name)
}

// CompareSection handles /compare/{section}/{name}
func (h *Handlers) CompareSection(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	h.serveCompare(w, r, section, name)
}

func (h *Handlers) serveCompare(w http.ResponseWriter, r *http.Request, section, name string) {
	if h.db == nil {
		h.renderError(w, http.StatusServiceUnavailable, "Database not initialized")
		return
	}

	result, err := h.db.Compare(name, section)
	if err != nil {
		h.renderError(w, http.StatusNotFound, fmt.Sprintf("No pages found for %s", name))
		return
	}

	data := template.CompareData{
		Title:   fmt.Sprintf("Compare %s - casman", name),
		Name:    name,
		Section: section,
		Result:  result,
	}

	h.renderTemplate(w, "compare.html", data, http.StatusOK)
}

// Whatis / Apropos handlers

// Whatis handles /whatis/{name}
func (h *Handlers) Whatis(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if h.db == nil {
		h.textResponse(w, name+" - database not available", http.StatusServiceUnavailable)
		return
	}

	results, err := h.db.Whatis(name)
	if err != nil || len(results) == 0 {
		h.textResponse(w, name+": nothing appropriate", http.StatusNotFound)
		return
	}

	var lines []string
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("%s(%s) - %s", r.Name, r.Section, r.Title))
	}

	h.textResponse(w, strings.Join(lines, "\n"), http.StatusOK)
}

// Apropos handles /apropos
func (h *Handlers) Apropos(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if h.db == nil || query == "" {
		h.textResponse(w, "apropos what?", http.StatusBadRequest)
		return
	}

	results, err := h.db.Apropos(query)
	if err != nil || len(results) == 0 {
		h.textResponse(w, query+": nothing appropriate", http.StatusNotFound)
		return
	}

	var lines []string
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("%s(%s) - %s", r.Name, r.Section, r.Title))
	}

	h.textResponse(w, strings.Join(lines, "\n"), http.StatusOK)
}

// API handlers

// APIStats handles /api/v1/stats
func (h *Handlers) APIStats(w http.ResponseWriter, r *http.Request) {
	var stats model.Stats
	if h.db != nil {
		stats, _ = h.db.GetStats()
	}
	h.jsonResponse(w, stats, http.StatusOK)
}

// APISections handles /api/v1/sections
func (h *Handlers) APISections(w http.ResponseWriter, r *http.Request) {
	sections := model.Sections
	if h.db != nil {
		sections, _ = h.db.GetSections()
	}
	h.jsonResponse(w, sections, http.StatusOK)
}

// APIPlatforms handles /api/v1/platforms
func (h *Handlers) APIPlatforms(w http.ResponseWriter, r *http.Request) {
	platforms := model.Platforms
	if h.db != nil {
		platforms, _ = h.db.GetPlatforms()
	}
	h.jsonResponse(w, platforms, http.StatusOK)
}

// APILanguages handles /api/v1/languages
func (h *Handlers) APILanguages(w http.ResponseWriter, r *http.Request) {
	languages := []map[string]interface{}{
		{"code": "en", "name": "English", "coverage": 100},
		{"code": "de", "name": "German", "coverage": 40},
		{"code": "fr", "name": "French", "coverage": 35},
		{"code": "ja", "name": "Japanese", "coverage": 30},
		{"code": "zh", "name": "Chinese", "coverage": 25},
		{"code": "es", "name": "Spanish", "coverage": 20},
		{"code": "ru", "name": "Russian", "coverage": 15},
		{"code": "pt", "name": "Portuguese", "coverage": 10},
	}
	h.jsonResponse(w, languages, http.StatusOK)
}

// APIPopular handles /api/v1/popular
func (h *Handlers) APIPopular(w http.ResponseWriter, r *http.Request) {
	var popular []model.ManPageSummary
	if h.db != nil {
		popular, _ = h.db.GetPopular(20)
	}
	h.jsonResponse(w, popular, http.StatusOK)
}

// APIManPage handles /api/v1/man/{name}
func (h *Handlers) APIManPage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.serveAPIManPage(w, r, "", "", name)
}

// APIManPageSection handles /api/v1/man/{section}/{name}
func (h *Handlers) APIManPageSection(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	h.serveAPIManPage(w, r, "", section, name)
}

// APIManPageOSSection handles /api/v1/man/{os}/{section}/{name}
func (h *Handlers) APIManPageOSSection(w http.ResponseWriter, r *http.Request) {
	osParam := chi.URLParam(r, "os")
	section := chi.URLParam(r, "section")
	name := chi.URLParam(r, "name")
	h.serveAPIManPage(w, r, osParam, section, name)
}

func (h *Handlers) serveAPIManPage(w http.ResponseWriter, r *http.Request, platform, section, name string) {
	if h.db == nil {
		h.jsonResponse(w, map[string]string{"error": "database not available"}, http.StatusServiceUnavailable)
		return
	}

	var page *model.ManPage
	var err error

	if platform != "" && section != "" {
		page, err = h.db.GetManPage(platform, section, name)
	} else if section != "" {
		page, err = h.db.GetManPageByName(name, section, "")
	} else {
		page, err = h.db.GetManPageByName(name, "", "")
	}

	if err != nil || page == nil {
		h.jsonResponse(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	h.jsonResponse(w, page, http.StatusOK)
}

// APISearch handles /api/v1/search
func (h *Handlers) APISearch(w http.ResponseWriter, r *http.Request) {
	rawQuery := r.URL.Query().Get("q")
	urlSection := r.URL.Query().Get("section")
	urlPlatform := r.URL.Query().Get("platform")
	pageStr := r.URL.Query().Get("page")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	parsed := ParseSearchQuery(rawQuery).MergeURLParams(urlSection, urlPlatform)

	if h.db == nil || parsed.Query == "" {
		h.jsonResponse(w, model.SearchResponse{
			Query:   rawQuery,
			Total:   0,
			Page:    page,
			Results: []model.SearchResult{},
		}, http.StatusOK)
		return
	}

	results, total, err := h.db.Search(parsed.Query, parsed.Section, parsed.Platform, page, 20)
	if err != nil {
		results = []model.SearchResult{}
		total = 0
	}

	h.jsonResponse(w, model.SearchResponse{
		Query:   rawQuery,
		Total:   total,
		Page:    page,
		Results: results,
	}, http.StatusOK)
}

// APIAutocomplete handles /api/v1/autocomplete
func (h *Handlers) APIAutocomplete(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if h.db == nil || len(query) < 2 {
		h.jsonResponse(w, map[string]interface{}{"suggestions": []interface{}{}}, http.StatusOK)
		return
	}

	suggestions, err := h.db.Autocomplete(query, 10)
	if err != nil {
		suggestions = []model.ManPageSummary{}
	}

	h.jsonResponse(w, map[string]interface{}{"suggestions": suggestions}, http.StatusOK)
}

// APISuggest handles /api/v1/suggest
func (h *Handlers) APISuggest(w http.ResponseWriter, r *http.Request) {
	h.APIAutocomplete(w, r)
}

// APICompare handles /api/v1/compare/{name}
func (h *Handlers) APICompare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	section := r.URL.Query().Get("section")

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

// APIWhatis handles /api/v1/whatis/{name}
func (h *Handlers) APIWhatis(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if h.db == nil {
		h.jsonResponse(w, map[string]string{"error": "database not available"}, http.StatusServiceUnavailable)
		return
	}

	results, err := h.db.Whatis(name)
	if err != nil || len(results) == 0 {
		h.jsonResponse(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	h.jsonResponse(w, results, http.StatusOK)
}

// APIApropos handles /api/v1/apropos
func (h *Handlers) APIApropos(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if h.db == nil || query == "" {
		h.jsonResponse(w, map[string]interface{}{
			"query":   query,
			"results": []interface{}{},
		}, http.StatusOK)
		return
	}

	results, err := h.db.Apropos(query)
	if err != nil {
		results = []model.ManPageSummary{}
	}

	h.jsonResponse(w, map[string]interface{}{
		"query":   query,
		"results": results,
	}, http.StatusOK)
}

// APITLDR handles /api/v1/tldr/{name}
func (h *Handlers) APITLDR(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	section := r.URL.Query().Get("section")

	if h.db == nil {
		h.jsonResponse(w, map[string]string{"error": "database not available"}, http.StatusServiceUnavailable)
		return
	}

	tldr, err := h.db.GetTLDR(name, section)
	if err != nil {
		h.jsonResponse(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	h.jsonResponse(w, tldr, http.StatusOK)
}

// Feed handlers

// Feed handles /feed.xml
func (h *Handlers) Feed(w http.ResponseWriter, r *http.Request) {
	h.serveFeed(w, r, "", "")
}

// FeedPlatform handles /feed/{platform}.xml
func (h *Handlers) FeedPlatform(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	h.serveFeed(w, r, platform, "")
}

// FeedSection handles /feed/section/{section}.xml
func (h *Handlers) FeedSection(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	h.serveFeed(w, r, "", section)
}

func (h *Handlers) serveFeed(w http.ResponseWriter, r *http.Request, platform, section string) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	var entries []model.FeedEntry
	if h.db != nil {
		entries, _ = h.db.GetRecentPages(platform, section, 50)
	}

	// Build feed
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>casman - Man Page Updates</title>
  <subtitle>Universal man pages from BSD, macOS, Linux, and more</subtitle>
  <link href="` + baseURL + `/feed.xml" rel="self"/>
  <link href="` + baseURL + `/" rel="alternate"/>
  <id>` + baseURL + `/</id>
`)

	if len(entries) > 0 {
		sb.WriteString("  <updated>" + entries[0].UpdatedAt.Format("2006-01-02T15:04:05Z") + "</updated>\n")
	} else {
		sb.WriteString("  <updated>" + time.Now().Format("2006-01-02T15:04:05Z") + "</updated>\n")
	}

	for _, e := range entries {
		sb.WriteString("  <entry>\n")
		sb.WriteString("    <title>" + e.Name + "(" + e.Section + ") - " + e.Platform + "</title>\n")
		sb.WriteString("    <link href=\"" + baseURL + e.URL + "\"/>\n")
		sb.WriteString("    <id>" + baseURL + e.URL + "</id>\n")
		sb.WriteString("    <updated>" + e.UpdatedAt.Format("2006-01-02T15:04:05Z") + "</updated>\n")
		sb.WriteString("    <summary>" + e.Title + "</summary>\n")
		sb.WriteString("    <category term=\"" + e.Platform + "\"/>\n")
		sb.WriteString("    <category term=\"section-" + e.Section + "\"/>\n")
		sb.WriteString("  </entry>\n")
	}

	sb.WriteString("</feed>\n")

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String()))
}

// FeedJSON handles /feed.json
func (h *Handlers) FeedJSON(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	var entries []model.FeedEntry
	if h.db != nil {
		entries, _ = h.db.GetRecentPages("", "", 50)
	}

	items := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		items = append(items, map[string]interface{}{
			"id":             baseURL + e.URL,
			"url":            baseURL + e.URL,
			"title":          e.Name + "(" + e.Section + ") - " + e.Title,
			"summary":        e.Summary,
			"date_published": e.UpdatedAt.Format(time.RFC3339),
			"tags":           []string{e.Platform, "section-" + e.Section},
		})
	}

	h.jsonResponse(w, map[string]interface{}{
		"version":       "https://jsonfeed.org/version/1.1",
		"title":         "casman - Man Page Updates",
		"home_page_url": baseURL,
		"feed_url":      baseURL + "/feed.json",
		"items":         items,
	}, http.StatusOK)
}

// SEO handlers

// Sitemap handles /sitemap.xml
func (h *Handlers) Sitemap(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>` + baseURL + `/sitemap-pages.xml</loc>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-sections.xml</loc>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-platforms.xml</loc>
  </sitemap>
</sitemapindex>
`))
}

// SitemapPages handles /sitemap-pages.xml
func (h *Handlers) SitemapPages(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)

	if h.db != nil {
		pages, _ := h.db.GetAllPageURLs()
		for _, p := range pages {
			sb.WriteString("  <url>\n")
			sb.WriteString("    <loc>" + baseURL + p.URL + "</loc>\n")
			sb.WriteString("    <changefreq>monthly</changefreq>\n")
			sb.WriteString("    <priority>0.8</priority>\n")
			sb.WriteString("  </url>\n")
		}
	}

	sb.WriteString("</urlset>\n")

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String()))
}

// SitemapSections handles /sitemap-sections.xml
func (h *Handlers) SitemapSections(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)

	// Add section browse pages
	for _, s := range model.Sections {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + baseURL + "/browse/" + s.ID + "</loc>\n")
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("    <priority>0.6</priority>\n")
		sb.WriteString("  </url>\n")
	}

	// Add platform browse pages
	for _, p := range model.Platforms {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + baseURL + "/browse/os/" + p.ID + "</loc>\n")
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("    <priority>0.6</priority>\n")
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>\n")

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String()))
}

// Robots handles /robots.txt
func (h *Handlers) Robots(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}
	h.textResponse(w, `User-agent: *
Allow: /

Sitemap: `+baseURL+`/sitemap.xml

Crawl-delay: 1

Disallow: /admin/
Disallow: /api/
Allow: /api/v1/openapi`, http.StatusOK)
}

// SecurityTxt handles /.well-known/security.txt per RFC 9116.
// Per AI.md: ALL projects MUST serve a valid security.txt file.
func (h *Handlers) SecurityTxt(w http.ResponseWriter, r *http.Request) {
	// Calculate expiry (1 year from now)
	expiry := time.Now().AddDate(1, 0, 0).Format(time.RFC3339)

	// Get security contact from config or use default
	contact := h.cfg.Server.FQDN
	if contact == "" {
		contact = r.Host
	}
	// Strip protocol if present
	contact = strings.TrimPrefix(contact, "http://")
	contact = strings.TrimPrefix(contact, "https://")

	h.textResponse(w, `Contact: mailto:security@`+contact+`
Expires: `+expiry+`
Preferred-Languages: en
Canonical: https://`+contact+`/.well-known/security.txt
`, http.StatusOK)
}

// Manifest handles /manifest.json for PWA support per AI.md PART 16.
func (h *Handlers) Manifest(w http.ResponseWriter, r *http.Request) {
	name := h.cfg.Server.Branding.Title
	if name == "" {
		name = "casman"
	}
	shortName := name
	if len(shortName) > 12 {
		shortName = shortName[:12]
	}

	description := h.cfg.Server.Branding.Description
	if description == "" {
		description = "Universal man pages from BSD, macOS, Linux, and more"
	}

	h.jsonResponse(w, map[string]interface{}{
		"name":             name,
		"short_name":       shortName,
		"description":      description,
		"start_url":        "/",
		"display":          "standalone",
		"background_color": "#1a1a2e",
		"theme_color":      "#e94560",
		"icons": []map[string]interface{}{
			{"src": "/static/icon-192.png", "sizes": "192x192", "type": "image/png"},
			{"src": "/static/icon-512.png", "sizes": "512x512", "type": "image/png"},
		},
	}, http.StatusOK)
}

// Favicon handles /favicon.ico - serves embedded default or redirects to static.
func (h *Handlers) Favicon(w http.ResponseWriter, r *http.Request) {
	// Redirect to SVG favicon in static assets
	http.Redirect(w, r, "/static/favicon.svg", http.StatusMovedPermanently)
}

// ServiceWorker handles /service-worker.js for PWA per AI.md PART 16.
func (h *Handlers) ServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`// casman Service Worker for PWA support
const CACHE_NAME = 'casman-v1';
const STATIC_ASSETS = [
  '/',
  '/static/style.css',
  '/static/favicon.svg',
  '/manifest.json'
];

// Install event - cache static assets
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => {
      return cache.addAll(STATIC_ASSETS);
    })
  );
  self.skipWaiting();
});

// Activate event - clean old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys => {
      return Promise.all(
        keys.filter(key => key !== CACHE_NAME)
            .map(key => caches.delete(key))
      );
    })
  );
  self.clients.claim();
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', event => {
  // Skip non-GET requests
  if (event.request.method !== 'GET') return;

  // Skip API requests (always fresh)
  if (event.request.url.includes('/api/')) return;

  event.respondWith(
    caches.match(event.request).then(cached => {
      return cached || fetch(event.request).then(response => {
        // Cache successful responses
        if (response.ok && response.type === 'basic') {
          const clone = response.clone();
          caches.open(CACHE_NAME).then(cache => {
            cache.put(event.request, clone);
          });
        }
        return response;
      });
    })
  );
});
`))
}

// APIRoot handles /api/v1/ - API root endpoint per AI.md PART 14.
func (h *Handlers) APIRoot(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.Server.FQDN
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}

	h.jsonResponse(w, map[string]interface{}{
		"name":        "casman",
		"description": "Universal Man Page API",
		"version":     h.version,
		"endpoints": map[string]string{
			"health":       baseURL + "/api/v1/healthz",
			"stats":        baseURL + "/api/v1/stats",
			"search":       baseURL + "/api/v1/search",
			"man":          baseURL + "/api/v1/man/{name}",
			"sections":     baseURL + "/api/v1/sections",
			"platforms":    baseURL + "/api/v1/platforms",
			"autocomplete": baseURL + "/api/v1/autocomplete",
			"openapi":      baseURL + "/api/v1/openapi",
		},
		"documentation": baseURL + "/openapi",
	}, http.StatusOK)
}

// OpenAPI handlers

// OpenAPIUI handles /openapi
func (h *Handlers) OpenAPIUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>casman API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: "/api/v1/openapi",
            dom_id: '#swagger-ui',
        });
    </script>
</body>
</html>
`))
}

// OpenAPISpec handles /openapi.json and /api/v1/openapi
func (h *Handlers) OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "casman API",
			"description": "Universal Man Page API",
			"version":     h.version,
		},
		"servers": []map[string]interface{}{
			{"url": h.cfg.Server.FQDN, "description": "Production server"},
		},
		"paths": map[string]interface{}{
			"/api/v1/healthz": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Healthy"},
					},
				},
			},
			"/api/v1/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get statistics",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Statistics"},
					},
				},
			},
			"/api/v1/search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Search man pages",
					"parameters": []map[string]interface{}{
						{"name": "q", "in": "query", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "section", "in": "query", "schema": map[string]string{"type": "string"}},
						{"name": "platform", "in": "query", "schema": map[string]string{"type": "string"}},
						{"name": "page", "in": "query", "schema": map[string]string{"type": "integer"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Search results"},
					},
				},
			},
			"/api/v1/man/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get man page by name",
					"parameters": []map[string]interface{}{
						{"name": "name", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Man page"},
						"404": map[string]interface{}{"description": "Not found"},
					},
				},
			},
			"/api/v1/man/{section}/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get man page by section and name",
					"parameters": []map[string]interface{}{
						{"name": "section", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "name", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Man page"},
						"404": map[string]interface{}{"description": "Not found"},
					},
				},
			},
			"/api/v1/man/{os}/{section}/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get man page by OS, section, and name",
					"parameters": []map[string]interface{}{
						{"name": "os", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "section", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "name", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Man page"},
						"404": map[string]interface{}{"description": "Not found"},
					},
				},
			},
		},
	}
	h.jsonResponse(w, spec, http.StatusOK)
}

// StaticFiles returns a handler for serving embedded static files.
func (h *Handlers) StaticFiles() http.Handler {
	subFS, err := fs.Sub(static.Files, ".")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Static files not available", http.StatusInternalServerError)
		})
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(subFS)))
}
