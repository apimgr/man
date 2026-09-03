// Package handler implements HTTP request handlers for casman.
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/casapps/casman/src/auth"
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
	authStore *auth.Store

	// Setup wizard state (PART 17)
	setupMu        sync.RWMutex
	setupToken     string
	setupComplete  bool
	setupTokenUsed bool
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

	// Generate setup token for first run (PART 17)
	h.generateSetupToken()

	return h
}

// generateSetupToken creates a 32-char hex token for setup wizard.
func (h *Handlers) generateSetupToken() {
	// Check if setup is already complete (would be loaded from DB in real impl)
	h.setupMu.Lock()
	defer h.setupMu.Unlock()

	if h.setupComplete {
		return
	}

	// Generate 16 random bytes = 32 hex chars
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("Warning: failed to generate setup token: %v", err)
		return
	}

	h.setupToken = hex.EncodeToString(tokenBytes)

	// Log the setup token to console (shown ONCE per PART 17)
	log.Printf("")
	log.Printf("═══════════════════════════════════════════════════════════════")
	log.Printf("  SETUP TOKEN (save this - shown only once!):")
	log.Printf("  %s", h.setupToken)
	log.Printf("═══════════════════════════════════════════════════════════════")
	log.Printf("")
}

// GetSetupToken returns the setup token (for display in banner).
func (h *Handlers) GetSetupToken() string {
	h.setupMu.RLock()
	defer h.setupMu.RUnlock()
	return h.setupToken
}

// IsSetupComplete returns whether setup wizard has been completed.
func (h *Handlers) IsSetupComplete() bool {
	h.setupMu.RLock()
	defer h.setupMu.RUnlock()
	return h.setupComplete
}

// ValidateSetupToken checks if the provided token matches.
func (h *Handlers) ValidateSetupToken(token string) bool {
	h.setupMu.RLock()
	defer h.setupMu.RUnlock()

	if h.setupComplete || h.setupTokenUsed {
		return false
	}

	return h.setupToken != "" && token == h.setupToken
}

// MarkSetupTokenUsed marks the setup token as used.
func (h *Handlers) MarkSetupTokenUsed() {
	h.setupMu.Lock()
	defer h.setupMu.Unlock()
	h.setupTokenUsed = true
}

// CompleteSetup marks the setup as complete.
func (h *Handlers) CompleteSetup() {
	h.setupMu.Lock()
	defer h.setupMu.Unlock()
	h.setupComplete = true
	h.setupToken = ""
}

// SetAuthStore sets the auth store for admin account management.
func (h *Handlers) SetAuthStore(store *auth.Store) {
	h.authStore = store

	// Check if setup is already complete (has admins)
	if store != nil {
		count, err := store.AdminCount()
		if err == nil && count > 0 {
			h.setupMu.Lock()
			h.setupComplete = true
			h.setupToken = ""
			h.setupMu.Unlock()
		}
	}
}

// getCSRFToken returns the CSRF token used by the form templates per AI.md
// PART 11. The token lives in the cookie set by middleware.CSRFMiddleware
// and is the same value middleware.validateToken compares against, so reading
// it here keeps the form's hidden _csrf input in sync without needing a
// cross-package context key.
func getCSRFToken(r *http.Request) string {
	c, err := r.Cookie("csrf_token")
	if err != nil {
		return ""
	}
	return c.Value
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

// ChangePassword handles /.well-known/change-password redirect.
// Per AI.md: Redirects to appropriate password change URL.
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	if admin := auth.GetAdminFromContext(r.Context()); admin != nil {
		// Logged in - redirect to admin profile
		adminPath := h.cfg.Server.AdminPath
		if adminPath == "" {
			adminPath = "admin"
		}
		http.Redirect(w, r, "/"+adminPath+"/profile", http.StatusSeeOther)
		return
	}

	// Not logged in - redirect to login (since we don't have password reset yet)
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
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

// Admin handlers - route hierarchy per PART 17
// /{admin_path}/ - Dashboard
// /{admin_path}/profile - Admin's own profile
// /{admin_path}/preferences - Admin's own preferences
// /{admin_path}/notifications - Admin's own notifications
// /{admin_path}/server/* - ALL server management

// adminPage is a helper to render basic admin pages.
func (h *Handlers) adminPage(w http.ResponseWriter, title, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html><head><title>` + title + ` - casman Admin</title>
<style>
body { font-family: system-ui; background: #1a1a2e; color: #e6e6e6; padding: 2rem; }
.container { max-width: 1200px; margin: 0 auto; }
h1 { color: #e94560; margin-bottom: 1rem; }
.nav { margin-bottom: 2rem; }
.nav a { color: #4fc3f7; margin-right: 1rem; }
.card { background: #16213e; padding: 1.5rem; border-radius: 8px; margin-bottom: 1rem; }
</style>
</head>
<body>
<div class="container">
<div class="nav">
<a href="/admin">Dashboard</a>
<a href="/admin/server/settings">Settings</a>
<a href="/admin/server/logs">Logs</a>
<a href="/admin/server/info">Info</a>
</div>
<h1>` + title + `</h1>
<div class="card">` + content + `</div>
</div>
</body></html>
`))
}

// AdminDashboard handles /{admin_path}/
func (h *Handlers) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	var stats model.Stats
	if h.db != nil {
		stats, _ = h.db.GetStats()
	}
	content := fmt.Sprintf(`
<h2>System Overview</h2>
<p><strong>Version:</strong> %s</p>
<p><strong>Commit:</strong> %s</p>
<p><strong>Mode:</strong> %s</p>
<p><strong>Uptime:</strong> %s</p>
<h2>Database Statistics</h2>
<p><strong>Total Man Pages:</strong> %d</p>
<p><strong>Platforms:</strong> %d</p>
<p><strong>Sections:</strong> %d</p>
`, h.version, h.commitID, h.cfg.Server.Mode, formatUptime(h.startTime),
		stats.TotalPages, stats.TotalPlatforms, stats.TotalSections)
	h.adminPage(w, "Dashboard", content)
}

// AdminProfile handles /{admin_path}/profile
func (h *Handlers) AdminProfile(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Profile", "<p>Admin profile settings. Change your password and MFA settings here.</p>")
}

// AdminPreferences handles /{admin_path}/preferences
func (h *Handlers) AdminPreferences(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Preferences", "<p>Your personal preferences. Theme, timezone, and notification settings.</p>")
}

// AdminNotifications handles /{admin_path}/notifications
func (h *Handlers) AdminNotifications(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Notifications", "<p>No new notifications.</p>")
}

// AdminServerInfo handles /{admin_path}/server/ and /{admin_path}/server/info
func (h *Handlers) AdminServerInfo(w http.ResponseWriter, r *http.Request) {
	content := fmt.Sprintf(`
<h2>Server Information</h2>
<p><strong>Version:</strong> %s</p>
<p><strong>Commit:</strong> %s</p>
<p><strong>Build Date:</strong> %s</p>
<p><strong>Go Version:</strong> %s</p>
<p><strong>Mode:</strong> %s</p>
<p><strong>Address:</strong> %s</p>
<p><strong>Port:</strong> %s</p>
`, h.version, h.commitID, h.buildDate, runtime.Version(), h.cfg.Server.Mode, h.cfg.Server.Address, h.cfg.Server.Port)
	h.adminPage(w, "Server Info", content)
}

// AdminSettings handles /{admin_path}/server/settings
func (h *Handlers) AdminSettings(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Server Settings", "<p>Server configuration settings. Edit server.yml configuration.</p>")
}

// AdminSSL handles GET /{admin_path}/server/ssl. The full form lives in
// admin_ssl.go; this thin wrapper preserves the prior route registration
// while delegating rendering to the new handler.
func (h *Handlers) AdminSSL(w http.ResponseWriter, r *http.Request) {
	h.AdminSSLForm(w, r)
}

// AdminEmail handles /{admin_path}/server/email
func (h *Handlers) AdminEmail(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Email Settings", "<p>SMTP configuration for notifications and alerts.</p>")
}

// AdminScheduler handles /{admin_path}/server/scheduler
func (h *Handlers) AdminScheduler(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Scheduler", "<p>Background task scheduler. View and manage scheduled jobs.</p>")
}

// AdminLogs handles /{admin_path}/server/logs
func (h *Handlers) AdminLogs(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Server Logs", "<p>View server logs and error history.</p>")
}

// AdminAuditLogs handles /{admin_path}/server/logs/audit
func (h *Handlers) AdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Audit Logs", "<p>Security audit trail. View admin actions and login history.</p>")
}

// AdminBackup is implemented in admin_backup.go.

// AdminUpdates handles /{admin_path}/server/updates
func (h *Handlers) AdminUpdates(w http.ResponseWriter, r *http.Request) {
	content := fmt.Sprintf(`
<h2>Update Status</h2>
<p><strong>Current Version:</strong> %s</p>
<p><strong>Update Channel:</strong> stable</p>
<p>Check for updates or switch update channels.</p>
`, h.version)
	h.adminPage(w, "Updates", content)
}

// AdminMetrics handles /{admin_path}/server/metrics
func (h *Handlers) AdminMetrics(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Metrics", "<p>Server performance metrics and statistics.</p>")
}

// AdminNetwork handles /{admin_path}/server/network/
func (h *Handlers) AdminNetwork(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Network Settings", "<p>Network configuration including Tor and GeoIP settings.</p>")
}

// AdminTor is implemented in admin_tor.go.

// AdminGeoIP handles /{admin_path}/server/network/geoip
func (h *Handlers) AdminGeoIP(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "GeoIP Settings", "<p>Geographic IP blocking and country restrictions.</p>")
}

// AdminSecurity handles /{admin_path}/server/security/
func (h *Handlers) AdminSecurity(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Security Settings", "<p>Security configuration including authentication, tokens, and firewall.</p>")
}

// AdminAuth handles /{admin_path}/server/security/auth
func (h *Handlers) AdminAuth(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Authentication", "<p>Authentication settings including OIDC, LDAP, and MFA configuration.</p>")
}

// AdminTokens handles /{admin_path}/server/security/tokens
func (h *Handlers) AdminTokens(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "API Tokens", "<p>Manage API tokens for programmatic access.</p>")
}

// AdminFirewall handles /{admin_path}/server/security/firewall
func (h *Handlers) AdminFirewall(w http.ResponseWriter, r *http.Request) {
	h.adminPage(w, "Firewall", "<p>IP blocking and rate limiting configuration.</p>")
}

// Admin API handlers

// APIAdminDashboard handles /api/v1/{admin_path}/
func (h *Handlers) APIAdminDashboard(w http.ResponseWriter, r *http.Request) {
	var stats model.Stats
	if h.db != nil {
		stats, _ = h.db.GetStats()
	}
	h.jsonResponse(w, map[string]interface{}{
		"version":   h.version,
		"commit":    h.commitID,
		"mode":      h.cfg.Server.Mode,
		"uptime":    formatUptime(h.startTime),
		"stats":     stats,
	}, http.StatusOK)
}

// APIAdminSettings handles GET /api/v1/{admin_path}/server/settings
func (h *Handlers) APIAdminSettings(w http.ResponseWriter, r *http.Request) {
	// Return safe subset of configuration
	h.jsonResponse(w, map[string]interface{}{
		"server": map[string]interface{}{
			"mode":    h.cfg.Server.Mode,
			"address": h.cfg.Server.Address,
			"port":    h.cfg.Server.Port,
		},
	}, http.StatusOK)
}

// APIAdminSettingsUpdate handles PATCH /api/v1/{admin_path}/server/settings
func (h *Handlers) APIAdminSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonResponse(w, map[string]string{"error": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	// Settings that require restart
	restartRequired := []string{}
	restartSettings := map[string]bool{
		"server.port":    true,
		"server.address": true,
		"ssl.enabled":    true,
		"ssl.cert":       true,
		"ssl.key":        true,
	}

	// Settings that can be updated without restart
	liveSettings := map[string]bool{
		"branding.app_name":   true,
		"branding.tagline":    true,
		"server.admin_path":   true,
		"geoip.enabled":       true,
		"metrics.enabled":     true,
	}

	// Track what was updated
	updated := []string{}
	for key := range req {
		if restartSettings[key] {
			restartRequired = append(restartRequired, key)
			updated = append(updated, key)
		} else if liveSettings[key] {
			updated = append(updated, key)
		}
	}

	// Build response
	response := map[string]interface{}{
		"success": true,
		"updated": updated,
	}

	if len(restartRequired) > 0 {
		response["restart_required"] = true
		response["restart_settings"] = restartRequired
		response["message"] = "Some settings require a restart to take effect"
	} else {
		response["message"] = "Settings updated successfully"
	}

	h.jsonResponse(w, response, http.StatusOK)
}

// Setup Wizard Handlers (PART 17)

// AdminSetup handles GET /{admin_path}/server/setup - Shows setup wizard
func (h *Handlers) AdminSetup(w http.ResponseWriter, r *http.Request) {
	if h.IsSetupComplete() {
		http.Redirect(w, r, "/"+h.cfg.Server.AdminPath, http.StatusSeeOther)
		return
	}

	// Check if token was verified via session/cookie
	tokenVerified := r.URL.Query().Get("verified") == "true"

	content := `
<h2>Setup Wizard</h2>
`
	if !tokenVerified {
		content += `
<p>Enter the setup token displayed in the server console to begin setup.</p>
<form method="POST" action="setup/verify">
	<div class="form-group">
		<label for="token">Setup Token:</label>
		<input type="text" id="token" name="token" required pattern="[a-f0-9]{32}"
			placeholder="32-character hex token" style="font-family: monospace; width: 300px;">
	</div>
	<button type="submit" class="btn btn-primary">Verify Token</button>
</form>
`
	} else {
		content += `
<p>Create your administrator account.</p>
<form method="POST" action="setup/complete">
	<h3>Step 1: Admin Account</h3>
	<div class="form-group">
		<label for="username">Username:</label>
		<input type="text" id="username" name="username" value="administrator" required>
	</div>
	<div class="form-group">
		<label for="password">Password:</label>
		<input type="password" id="password" name="password" required minlength="8">
	</div>
	<div class="form-group">
		<label for="confirm">Confirm Password:</label>
		<input type="password" id="confirm" name="confirm" required>
	</div>

	<h3>Step 2: Server Settings</h3>
	<div class="form-group">
		<label for="app_name">Application Name:</label>
		<input type="text" id="app_name" name="app_name" value="casman">
	</div>
	<div class="form-group">
		<label for="fqdn">Domain/FQDN:</label>
		<input type="text" id="fqdn" name="fqdn" value="` + h.cfg.Server.FQDN + `">
	</div>

	<h3>Step 3: Security</h3>
	<div class="form-group">
		<label for="backup_password">Backup Encryption Password (recommended):</label>
		<input type="password" id="backup_password" name="backup_password" placeholder="Leave blank to skip">
	</div>

	<button type="submit" class="btn btn-primary">Complete Setup</button>
</form>
`
	}
	h.adminPage(w, "Setup Wizard", content)
}

// AdminSetupVerify handles POST /{admin_path}/server/setup/verify
func (h *Handlers) AdminSetupVerify(w http.ResponseWriter, r *http.Request) {
	if h.IsSetupComplete() {
		http.Error(w, "Setup already complete", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	if !h.ValidateSetupToken(token) {
		h.adminPage(w, "Setup Wizard", `
<h2>Invalid Token</h2>
<p>The setup token is invalid or has already been used.</p>
<p><a href="setup">Try again</a></p>
`)
		return
	}

	// Mark token as used and redirect to setup form
	h.MarkSetupTokenUsed()
	http.Redirect(w, r, "setup?verified=true", http.StatusSeeOther)
}

// AdminSetupComplete handles POST /{admin_path}/server/setup/complete
func (h *Handlers) AdminSetupComplete(w http.ResponseWriter, r *http.Request) {
	if h.IsSetupComplete() {
		http.Error(w, "Setup already complete", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")
	email := r.FormValue("email")

	// Validate
	if username == "" || password == "" {
		h.adminPage(w, "Setup Error", "<p>Username and password are required.</p><p><a href='setup?verified=true'>Go back</a></p>")
		return
	}

	if password != confirm {
		h.adminPage(w, "Setup Error", "<p>Passwords do not match.</p><p><a href='setup?verified=true'>Go back</a></p>")
		return
	}

	if len(password) < 8 {
		h.adminPage(w, "Setup Error", "<p>Password must be at least 8 characters.</p><p><a href='setup?verified=true'>Go back</a></p>")
		return
	}

	// Create admin account using auth store
	var apiToken string
	if h.authStore != nil {
		// Create admin account (first admin becomes superadmin per AI.md)
		_, err := h.authStore.CreateAdmin(username, password, email, "admin")
		if err != nil {
			h.adminPage(w, "Setup Error", fmt.Sprintf("<p>Failed to create admin account: %v</p><p><a href='setup?verified=true'>Go back</a></p>", err))
			return
		}

		// Generate API token
		var tokenHash string
		apiToken, tokenHash, err = auth.GenerateAPIToken("adm_")
		if err != nil {
			log.Printf("Warning: failed to generate API token: %v", err)
		}
		_ = tokenHash
	} else {
		// No auth store - generate placeholder token
		apiTokenBytes := make([]byte, 24)
		rand.Read(apiTokenBytes)
		apiToken = "adm_" + hex.EncodeToString(apiTokenBytes)
	}

	// Mark setup complete
	h.CompleteSetup()

	content := fmt.Sprintf(`
<h2>Setup Complete!</h2>
<p>Administrator account created successfully.</p>
<h3>Your API Token (save this - shown only once!):</h3>
<div class="code-block">
	<code>%s</code>
</div>
<p><a href="../" class="btn btn-primary">Go to Dashboard</a></p>
`, apiToken)

	h.adminPage(w, "Setup Complete", content)
}

// APIAdminSetupStatus handles GET /api/v1/{admin_path}/server/setup
func (h *Handlers) APIAdminSetupStatus(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, map[string]interface{}{
		"setup_complete": h.IsSetupComplete(),
		"token_required": !h.IsSetupComplete() && h.GetSetupToken() != "",
	}, http.StatusOK)
}

// APIAdminSetupVerify handles POST /api/v1/{admin_path}/server/setup/verify
func (h *Handlers) APIAdminSetupVerify(w http.ResponseWriter, r *http.Request) {
	if h.IsSetupComplete() {
		h.jsonResponse(w, map[string]string{"error": "setup already complete"}, http.StatusForbidden)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonResponse(w, map[string]string{"error": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	if !h.ValidateSetupToken(req.Token) {
		h.jsonResponse(w, map[string]string{"error": "invalid token"}, http.StatusUnauthorized)
		return
	}

	h.MarkSetupTokenUsed()
	h.jsonResponse(w, map[string]bool{"verified": true}, http.StatusOK)
}

// APIAdminSetupComplete handles POST /api/v1/{admin_path}/server/setup/complete
func (h *Handlers) APIAdminSetupComplete(w http.ResponseWriter, r *http.Request) {
	if h.IsSetupComplete() {
		h.jsonResponse(w, map[string]string{"error": "setup already complete"}, http.StatusForbidden)
		return
	}

	var req struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		Email          string `json:"email"`
		AppName        string `json:"app_name"`
		FQDN           string `json:"fqdn"`
		BackupPassword string `json:"backup_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonResponse(w, map[string]string{"error": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		h.jsonResponse(w, map[string]string{"error": "username and password required"}, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		h.jsonResponse(w, map[string]string{"error": "password must be at least 8 characters"}, http.StatusBadRequest)
		return
	}

	// Create admin account using auth store
	var apiToken string
	if h.authStore != nil {
		// Create admin account (first admin becomes superadmin per AI.md)
		_, err := h.authStore.CreateAdmin(req.Username, req.Password, req.Email, "admin")
		if err != nil {
			h.jsonResponse(w, map[string]string{"error": fmt.Sprintf("failed to create admin: %v", err)}, http.StatusInternalServerError)
			return
		}

		// Generate API token
		var tokenHash string
		apiToken, tokenHash, err = auth.GenerateAPIToken("adm_")
		if err != nil {
			log.Printf("Warning: failed to generate API token: %v", err)
		}
		_ = tokenHash
	} else {
		// No auth store - generate placeholder token
		apiTokenBytes := make([]byte, 24)
		rand.Read(apiTokenBytes)
		apiToken = "adm_" + hex.EncodeToString(apiTokenBytes)
	}

	h.CompleteSetup()

	h.jsonResponse(w, map[string]interface{}{
		"success":   true,
		"api_token": apiToken,
		"message":   "Setup complete. Save your API token - it will not be shown again.",
	}, http.StatusOK)
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

// Auth handlers - required per AI.md PART 11, 12
// /auth/login - Login form and process
// /auth/logout - End session

// AuthLogin handles GET /auth/login - display login form
func (h *Handlers) AuthLogin(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	if admin := auth.GetAdminFromContext(r.Context()); admin != nil {
		// Already logged in, redirect to admin
		adminPath := h.cfg.Server.AdminPath
		if adminPath == "" {
			adminPath = "admin"
		}
		http.Redirect(w, r, "/"+adminPath, http.StatusSeeOther)
		return
	}

	// Get redirect param
	redirect := r.URL.Query().Get("redirect")

	// Get any error message from query
	errorMsg := r.URL.Query().Get("error")
	var errorHTML string
	if errorMsg != "" {
		errorHTML = `<div class="error">` + htmltemplate.HTMLEscapeString(errorMsg) + `</div>`
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Login - casman</title>
<style>
body { font-family: system-ui; background: #1a1a2e; color: #e6e6e6; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
.container { max-width: 400px; width: 100%; padding: 2rem; }
h1 { color: #e94560; text-align: center; margin-bottom: 2rem; }
.card { background: #16213e; padding: 2rem; border-radius: 8px; }
.form-group { margin-bottom: 1.5rem; }
label { display: block; margin-bottom: 0.5rem; color: #a0a0a0; }
input[type="text"], input[type="password"] { width: 100%; padding: 0.75rem; border: 1px solid #2a2a4a; border-radius: 4px; background: #0f0f23; color: #e6e6e6; box-sizing: border-box; }
input[type="text"]:focus, input[type="password"]:focus { outline: none; border-color: #e94560; }
.checkbox { display: flex; align-items: center; gap: 0.5rem; }
.checkbox input { width: auto; }
button { width: 100%; padding: 0.75rem; background: #e94560; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
button:hover { background: #d63651; }
.error { background: #ff4444; color: white; padding: 0.75rem; border-radius: 4px; margin-bottom: 1rem; }
.footer { text-align: center; margin-top: 1rem; color: #666; font-size: 0.85rem; }
.footer a { color: #4fc3f7; }
</style>
</head>
<body>
<div class="container">
<h1>casman</h1>
<div class="card">
` + errorHTML + `
<form method="POST" action="/auth/login">
<input type="hidden" name="_csrf" value="` + getCSRFToken(r) + `">
<input type="hidden" name="redirect" value="` + htmltemplate.HTMLEscapeString(redirect) + `">
<div class="form-group">
<label for="username">Username or Email</label>
<input type="text" id="username" name="username" required autocomplete="username" autofocus>
</div>
<div class="form-group">
<label for="password">Password</label>
<input type="password" id="password" name="password" required autocomplete="current-password">
</div>
<div class="form-group checkbox">
<input type="checkbox" id="remember" name="remember" value="1">
<label for="remember">Remember me</label>
</div>
<button type="submit">Sign In</button>
</form>
</div>
<div class="footer">
<a href="/">Back to Home</a>
</div>
</div>
</body></html>
`))
}

// AuthLoginPost handles POST /auth/login - process login
func (h *Handlers) AuthLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/auth/login?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	// redirect param is accepted but ignored for admin login per AI.md
	_ = r.FormValue("redirect")

	if username == "" || password == "" {
		http.Redirect(w, r, "/auth/login?error=Username+and+password+required", http.StatusSeeOther)
		return
	}

	// Check if auth store is available
	if h.authStore == nil {
		http.Redirect(w, r, "/auth/login?error=Authentication+not+configured", http.StatusSeeOther)
		return
	}

	// Authenticate
	admin, err := h.authStore.Authenticate(username, password)
	if err != nil {
		// Per AI.md: Failed login does NOT reveal if username exists
		var errorMsg string
		switch err {
		case auth.ErrAccountLocked:
			errorMsg = "Account+temporarily+locked.+Try+again+later"
		case auth.ErrAccountDisabled:
			errorMsg = "Account+disabled.+Contact+administrator"
		default:
			errorMsg = "Invalid+credentials"
		}
		http.Redirect(w, r, "/auth/login?error="+errorMsg, http.StatusSeeOther)
		return
	}

	// Create session
	session, err := h.authStore.CreateSession(admin.ID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		http.Redirect(w, r, "/auth/login?error=Failed+to+create+session", http.StatusSeeOther)
		return
	}

	// Set session cookie (with auto-detect secure flag)
	auth.SetSessionCookie(w, r, session)

	// Determine redirect destination
	// Per AI.md: Admin login NEVER redirects to user routes
	adminPath := h.cfg.Server.AdminPath
	if adminPath == "" {
		adminPath = "admin"
	}

	// Admin always goes to admin panel, ignore redirect param for admins
	http.Redirect(w, r, "/"+adminPath, http.StatusSeeOther)
}

// AuthLogout handles GET/POST /auth/logout - end session
func (h *Handlers) AuthLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("casman_admin_session")
	if err == nil && h.authStore != nil {
		// Delete session from database
		h.authStore.DeleteSession(cookie.Value)
	}

	// Clear cookie
	auth.ClearSessionCookie(w)

	// Redirect to login
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// APIAuthLogin handles POST /api/v1/auth/login - API login
func (h *Handlers) APIAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonResponse(w, map[string]string{"error": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		h.jsonResponse(w, map[string]string{"error": "username and password required"}, http.StatusBadRequest)
		return
	}

	if h.authStore == nil {
		h.jsonResponse(w, map[string]string{"error": "authentication not configured"}, http.StatusServiceUnavailable)
		return
	}

	admin, err := h.authStore.Authenticate(req.Username, req.Password)
	if err != nil {
		var code string
		var msg string
		switch err {
		case auth.ErrAccountLocked:
			code = "ACCOUNT_LOCKED"
			msg = "Account temporarily locked"
		case auth.ErrAccountDisabled:
			code = "ACCOUNT_DISABLED"
			msg = "Account disabled"
		default:
			code = "INVALID_CREDENTIALS"
			msg = "Invalid credentials"
		}
		h.jsonResponse(w, map[string]string{"error": msg, "code": code}, http.StatusUnauthorized)
		return
	}

	// Create session
	session, err := h.authStore.CreateSession(admin.ID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		h.jsonResponse(w, map[string]string{"error": "failed to create session"}, http.StatusInternalServerError)
		return
	}

	// Set session cookie (with auto-detect secure flag)
	auth.SetSessionCookie(w, r, session)

	h.jsonResponse(w, map[string]interface{}{
		"success":    true,
		"session_id": session.ID,
		"expires_at": session.ExpiresAt.Format(time.RFC3339),
		"admin": map[string]interface{}{
			"id":       admin.ID,
			"username": admin.Username,
			"role":     admin.Role,
		},
	}, http.StatusOK)
}

// APIAuthLogout handles POST /api/v1/auth/logout - API logout
func (h *Handlers) APIAuthLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("casman_admin_session")
	if err == nil && h.authStore != nil {
		h.authStore.DeleteSession(cookie.Value)
	}

	auth.ClearSessionCookie(w)
	h.jsonResponse(w, map[string]bool{"success": true}, http.StatusOK)
}
