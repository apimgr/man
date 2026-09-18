package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/casapps/casman/src/config"
	"github.com/casapps/casman/src/server/model"
	"github.com/casapps/casman/src/server/store"
	"github.com/casapps/casman/src/server/template"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	tmpl, err := template.New()
	if err != nil {
		t.Fatalf("template.New: %v", err)
	}
	cfg := &config.Config{}
	cfg.Server.FQDN = "example.com"
	cfg.Server.Mode = "production"
	cfg.Server.Branding.Title = "casman"
	cfg.Server.Branding.Description = "Universal man pages"

	return &Handlers{
		cfg:       cfg,
		version:   "1.0.0",
		commitID:  "abc123",
		buildDate: "2024-01-01",
		startTime: time.Now().Add(-90 * time.Minute),
		tmpl:      tmpl,
	}
}

func newTestHandlersWithDB(t *testing.T) *Handlers {
	t.Helper()
	h := newTestHandlers(t)
	dir := t.TempDir()
	db, err := store.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	page := &model.ManPage{
		Name:            "ls",
		Section:         "1",
		Title:           "ls - list directory contents",
		Platform:        "linux",
		SourceFormat:    "groff",
		SourceRaw:       ".TH ls",
		ContentHTML:     "<p>ls content</p>",
		ContentText:     "ls content",
		ContentMarkdown: "# ls",
		ContentRaw:      ".TH ls",
		Synopsis:        "ls [OPTIONS]",
		Description:     "Lists directory contents",
		SearchText:      "ls list directory contents",
		SeeAlso: []model.SeeAlsoEntry{
			{Name: "cd", Section: "1"},
		},
	}
	if err := db.InsertManPage(page); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	h.db = db
	return h
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		delta time.Duration
		want  string
	}{
		{time.Minute * 5, "5m"},
		{time.Hour*2 + time.Minute*3, "2h 3m"},
		{time.Hour*25 + time.Minute*1, "1d 1h 1m"},
	}
	for _, c := range cases {
		got := formatUptime(time.Now().Add(-c.delta))
		if got != c.want {
			t.Errorf("formatUptime(-%v) = %q, want %q", c.delta, got, c.want)
		}
	}
}

func TestBuildHealthResponse_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	health := h.buildHealthResponse()
	if health.Status != "degraded" {
		t.Errorf("Status = %q, want degraded (nil db)", health.Status)
	}
	if health.Checks.Database != "error" {
		t.Errorf("Checks.Database = %q, want error", health.Checks.Database)
	}
	if health.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", health.Version)
	}
}

func TestAPIHealthz(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	h.APIHealthz(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "degraded" {
		t.Errorf("status field = %v, want degraded", body["status"])
	}
}

func TestHealthz_JSONAccept(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/healthz", nil)
	r.Header.Set("Accept", "application/json")
	h.Healthz(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q, want application/json", w.Header().Get("Content-Type"))
	}
}

func TestHealthz_TextAccept(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/healthz", nil)
	r.Header.Set("Accept", "text/plain")
	h.Healthz(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "casman") {
		t.Errorf("body = %q, want to contain casman", w.Body.String())
	}
}

func TestHealthz_HTMLDefault(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/healthz", nil)
	h.Healthz(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("Content-Type = %q, want text/html", w.Header().Get("Content-Type"))
	}
}

func TestHome_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h.Home(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestRobots(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/robots.txt", nil)
	h.Robots(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Sitemap: example.com/sitemap.xml") {
		t.Errorf("body = %q, missing expected sitemap line", w.Body.String())
	}
}

func TestSecurityTxt(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/.well-known/security.txt", nil)
	h.SecurityTxt(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Contact: mailto:security@example.com") {
		t.Errorf("body = %q, missing expected contact line", w.Body.String())
	}
}

func TestManifest(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/manifest.json", nil)
	h.Manifest(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["name"] != "casman" {
		t.Errorf("name = %v, want casman", body["name"])
	}
}

func TestFavicon(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/favicon.ico", nil)
	h.Favicon(w, r)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/static/favicon.svg" {
		t.Errorf("Location = %q, want /static/favicon.svg", loc)
	}
}

func TestServiceWorker(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/service-worker.js", nil)
	h.ServiceWorker(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "CACHE_NAME") {
		t.Errorf("body missing expected service worker content")
	}
}

func TestAPIRoot(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/", nil)
	h.APIRoot(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["name"] != "casman" {
		t.Errorf("name = %v, want casman", body["name"])
	}
}

func TestOpenAPIUI(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/openapi", nil)
	h.OpenAPIUI(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "swagger-ui") {
		t.Errorf("body missing swagger-ui content")
	}
}

func TestOpenAPISpec(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/openapi.json", nil)
	h.OpenAPISpec(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["openapi"] != "3.0.0" {
		t.Errorf("openapi = %v, want 3.0.0", body["openapi"])
	}
}

func TestAPIStats_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/stats", nil)
	h.APIStats(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPISections_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/sections", nil)
	h.APISections(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body) == 0 {
		t.Errorf("expected non-empty default sections")
	}
}

func TestAPIPlatforms_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/platforms", nil)
	h.APIPlatforms(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPILanguages(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/languages", nil)
	h.APILanguages(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "English") {
		t.Errorf("body missing expected language entry")
	}
}

func TestAPIPopular_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/popular", nil)
	h.APIPopular(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestAPIManPage_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.APIManPage(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestAPIManPageSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/1/ls", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "ls"})
	h.APIManPageSection(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestAPIManPageOSSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/linux/1/ls", nil)
	r = withChiParams(r, map[string]string{"os": "linux", "section": "1", "name": "ls"})
	h.APIManPageOSSection(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestAPISearch_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/search?q=ls", nil)
	h.APISearch(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["query"] != "ls" {
		t.Errorf("query = %v, want ls", body["query"])
	}
}

func TestAPISearch_EmptyQuery(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	h.APISearch(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPIAutocomplete_ShortQuery(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/autocomplete?q=l", nil)
	h.APIAutocomplete(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"suggestions": []`) {
		t.Errorf("expected empty suggestions, got %q", w.Body.String())
	}
}

func TestAPISuggest_DelegatesToAutocomplete(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/suggest?q=l", nil)
	h.APISuggest(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPICompare_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/compare/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.APICompare(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestAPIWhatis_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/whatis/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.APIWhatis(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestAPIApropos_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/apropos?q=list", nil)
	h.APIApropos(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPIApropos_EmptyQuery(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/apropos", nil)
	h.APIApropos(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPITLDR_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/tldr/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.APITLDR(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestFeed_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/feed.xml", nil)
	h.Feed(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<feed") {
		t.Errorf("body missing <feed> element")
	}
}

func TestFeedPlatform_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/feed/linux.xml", nil)
	r = withChiParams(r, map[string]string{"platform": "linux"})
	h.FeedPlatform(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestFeedSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/feed/section/1.xml", nil)
	r = withChiParams(r, map[string]string{"section": "1"})
	h.FeedSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestFeedJSON_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/feed.json", nil)
	h.FeedJSON(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["version"] != "https://jsonfeed.org/version/1.1" {
		t.Errorf("version = %v, want jsonfeed version", body["version"])
	}
}

func TestSitemap(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sitemap.xml", nil)
	h.Sitemap(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "sitemapindex") {
		t.Errorf("body missing sitemapindex element")
	}
}

func TestSitemapPages_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sitemap-pages.xml", nil)
	h.SitemapPages(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestSitemapSections(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sitemap-sections.xml", nil)
	h.SitemapSections(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "/browse/") {
		t.Errorf("body missing browse links")
	}
}

func TestSearch_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?q=ls", nil)
	h.Search(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestBrowse_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/browse", nil)
	h.Browse(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestBrowseSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/browse/1", nil)
	r = withChiParams(r, map[string]string{"section": "1"})
	h.BrowseSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestBrowseOS_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/browse/os/linux", nil)
	r = withChiParams(r, map[string]string{"os": "linux"})
	h.BrowseOS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestCompare_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/compare/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.Compare(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestCompareSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/compare/1/ls", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "ls"})
	h.CompareSection(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestWhatis_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/whatis/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.Whatis(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), "database not available") {
		t.Errorf("body = %q, want database not available message", w.Body.String())
	}
}

func TestApropos_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/apropos", nil)
	h.Apropos(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestManPage_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.ManPage(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestStaticFiles(t *testing.T) {
	h := newTestHandlers(t)
	handler := h.StaticFiles()
	if handler == nil {
		t.Fatal("StaticFiles() returned nil")
	}
}

func TestManPage_WithDB_HTML(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.ManPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ls content") {
		t.Errorf("body missing content: %s", w.Body.String())
	}
}

func TestManPage_WithDB_NotFound(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/nonexistent", nil)
	r = withChiParams(r, map[string]string{"name": "nonexistent"})
	h.ManPage(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestManPageWithFormat_Text(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/ls.txt", nil)
	r = withChiParams(r, map[string]string{"name": "ls", "ext": "txt"})
	h.ManPageWithFormat(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ls content") {
		t.Errorf("body = %q, want ls content", w.Body.String())
	}
}

func TestManPageWithFormat_Markdown(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/ls.md", nil)
	r = withChiParams(r, map[string]string{"name": "ls", "ext": "md"})
	h.ManPageWithFormat(w, r)

	if !strings.Contains(w.Body.String(), "# ls") {
		t.Errorf("body = %q, want markdown content", w.Body.String())
	}
}

func TestManPageWithFormat_Raw(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/ls.raw", nil)
	r = withChiParams(r, map[string]string{"name": "ls", "ext": "raw"})
	h.ManPageWithFormat(w, r)

	if !strings.Contains(w.Body.String(), ".TH ls") {
		t.Errorf("body = %q, want raw source", w.Body.String())
	}
}

func TestManPageWithFormat_JSON(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/ls.json", nil)
	r = withChiParams(r, map[string]string{"name": "ls", "ext": "json"})
	h.ManPageWithFormat(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var page model.ManPage
	if err := json.NewDecoder(w.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Name != "ls" {
		t.Errorf("Name = %q, want ls", page.Name)
	}
}

func TestManPageSection_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/1/ls", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "ls"})
	h.ManPageSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestManPageSectionWithFormat_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/1/ls.txt", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "ls", "ext": "txt"})
	h.ManPageSectionWithFormat(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestManPageOSSection_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/linux/1/ls", nil)
	r = withChiParams(r, map[string]string{"os": "linux", "section": "1", "name": "ls"})
	h.ManPageOSSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestManPageOSSectionWithFormat_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/man/linux/1/ls.txt", nil)
	r = withChiParams(r, map[string]string{"os": "linux", "section": "1", "name": "ls", "ext": "txt"})
	h.ManPageOSSectionWithFormat(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPIManPage_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.APIManPage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var page model.ManPage
	if err := json.NewDecoder(w.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Name != "ls" {
		t.Errorf("Name = %q, want ls", page.Name)
	}
}

func TestAPIManPage_WithDB_NotFound(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/nonexistent", nil)
	r = withChiParams(r, map[string]string{"name": "nonexistent"})
	h.APIManPage(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPIManPageSection_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/1/ls", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "ls"})
	h.APIManPageSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPIManPageOSSection_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/linux/1/ls", nil)
	r = withChiParams(r, map[string]string{"os": "linux", "section": "1", "name": "ls"})
	h.APIManPageOSSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestSearch_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?q=ls", nil)
	h.Search(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestBrowse_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/browse", nil)
	h.Browse(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestWhatis_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/whatis/ls", nil)
	r = withChiParams(r, map[string]string{"name": "ls"})
	h.Whatis(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestApropos_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/apropos?q=ls", nil)
	h.Apropos(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestAPIExportFormats(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/export/formats", nil)
	h.APIExportFormats(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestExportSection_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export/section/1.html", nil)
	r = withChiParams(r, map[string]string{"section": "1.html"})
	h.ExportSection(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestExportSection_NotFound(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export/section/9.html", nil)
	r = withChiParams(r, map[string]string{"section": "9.html"})
	h.ExportSection(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestExportSection_BadFormat(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export/section/1.bogus", nil)
	r = withChiParams(r, map[string]string{"section": "1.bogus"})
	h.ExportSection(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestExportSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export/section/1.html", nil)
	r = withChiParams(r, map[string]string{"section": "1.html"})
	h.ExportSection(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestExportPlatform_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export/platform/linux.epub", nil)
	r = withChiParams(r, map[string]string{"platform": "linux.epub"})
	h.ExportPlatform(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestExportPlatform_NotFound(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export/platform/plan9.pdf", nil)
	r = withChiParams(r, map[string]string{"platform": "plan9.pdf"})
	h.ExportPlatform(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestSplitExt(t *testing.T) {
	cases := map[string][2]string{
		"section.html": {"section", "html"},
		"noext":        {"noext", ""},
		".hidden":      {".hidden", ""},
	}
	for in, want := range cases {
		name, ext := splitExt(in)
		if name != want[0] || ext != want[1] {
			t.Errorf("splitExt(%q) = (%q, %q), want (%q, %q)", in, name, ext, want[0], want[1])
		}
	}
}

func TestAPIManPageLang_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/man/de/linux/1/ls", nil)
	r = withChiParams(r, map[string]string{"lang": "de", "os": "linux", "section": "1", "name": "ls"})
	h.APIManPageLang(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Content-Language"); got != "de" {
		t.Errorf("X-Content-Language = %q, want de", got)
	}
}

func TestAPICompareSection_NilDB(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/compare/1/ls", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "ls"})
	h.APICompareSection(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestAPICompareSection_NotFound(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/compare/1/missing", nil)
	r = withChiParams(r, map[string]string{"section": "1", "name": "missing"})
	h.APICompareSection(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestFeedCombined_WithDB(t *testing.T) {
	h := newTestHandlersWithDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/feed/linux/1.xml", nil)
	r = withChiParams(r, map[string]string{"platform": "linux", "section": "1.xml"})
	h.FeedCombined(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestSitemapPlatforms(t *testing.T) {
	h := newTestHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sitemap-platforms.xml", nil)
	h.SitemapPlatforms(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if !strings.Contains(w.Body.String(), "<urlset") {
		t.Error("expected urlset in body")
	}
}
