package store

import (
	"path/filepath"
	"testing"

	"github.com/casapps/casman/src/server/model"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func samplePage(name, section, platform string) *model.ManPage {
	return &model.ManPage{
		Name:            name,
		Section:         section,
		Title:           name + " - list directory contents",
		Platform:        platform,
		SourceFormat:    "groff",
		SourceRaw:       ".TH " + name,
		ContentHTML:     "<p>" + name + " content</p>",
		ContentText:     name + " content",
		ContentMarkdown: "# " + name,
		Synopsis:        name + " [OPTIONS]",
		Description:     "Lists directory contents for " + name,
		SearchText:      name + " list directory contents",
		SeeAlso: []model.SeeAlsoEntry{
			{Name: "cd", Section: "1"},
			{Name: "pwd"},
		},
	}
}

func TestNewCreatesSchema(t *testing.T) {
	db := newTestDB(t)
	if db.conn == nil {
		t.Fatal("conn is nil")
	}
}

func TestInsertAndGetManPage(t *testing.T) {
	db := newTestDB(t)
	page := samplePage("ls", "1", "linux")
	if err := db.InsertManPage(page); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}
	if page.ID == 0 {
		t.Fatal("expected non-zero ID after insert")
	}

	got, err := db.GetManPage("linux", "1", "ls")
	if err != nil {
		t.Fatalf("GetManPage: %v", err)
	}
	if got == nil || got.Name != "ls" {
		t.Fatalf("GetManPage = %+v, want name ls", got)
	}
	if len(got.SeeAlso) != 2 {
		t.Errorf("SeeAlso = %+v, want 2 entries", got.SeeAlso)
	}
}

func TestInsertManPage_UpdateOnConflict(t *testing.T) {
	db := newTestDB(t)
	page := samplePage("ls", "1", "linux")
	if err := db.InsertManPage(page); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}
	firstID := page.ID

	page2 := samplePage("ls", "1", "linux")
	page2.Title = "ls - updated title"
	if err := db.InsertManPage(page2); err != nil {
		t.Fatalf("InsertManPage (update): %v", err)
	}
	if page2.ID != firstID {
		t.Errorf("ID changed on update: got %d, want %d", page2.ID, firstID)
	}

	got, err := db.GetManPage("linux", "1", "ls")
	if err != nil {
		t.Fatalf("GetManPage: %v", err)
	}
	if got.Title != "ls - updated title" {
		t.Errorf("Title = %q, want updated title", got.Title)
	}
}

func TestGetManPage_NotFound(t *testing.T) {
	db := newTestDB(t)
	page, err := db.GetManPage("linux", "1", "nonexistent")
	if err == nil && page != nil {
		t.Fatalf("expected error or nil page, got %+v", page)
	}
}

func TestGetManPageByName(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("grep", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	got, err := db.GetManPageByName("grep", "", "")
	if err != nil {
		t.Fatalf("GetManPageByName: %v", err)
	}
	if got == nil || got.Name != "grep" {
		t.Fatalf("GetManPageByName = %+v, want name grep", got)
	}
}

func TestGetOtherPlatforms(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage linux: %v", err)
	}
	if err := db.InsertManPage(samplePage("ls", "1", "macos")); err != nil {
		t.Fatalf("InsertManPage macos: %v", err)
	}

	platforms, err := db.GetOtherPlatforms("ls", "1", "linux")
	if err != nil {
		t.Fatalf("GetOtherPlatforms: %v", err)
	}
	if len(platforms) != 1 || platforms[0] != "macos" {
		t.Errorf("GetOtherPlatforms = %v, want [macos]", platforms)
	}
}

func TestSearch(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	results, total, err := db.Search("directory", "", "", 1, 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total == 0 || len(results) == 0 {
		t.Errorf("Search returned no results: total=%d results=%+v", total, results)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	results, total, err := db.Search("zzz_no_such_term", "", "", 1, 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Errorf("Search = total=%d results=%+v, want empty", total, results)
	}
}

func TestAutocomplete(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	suggestions, err := db.Autocomplete("ls", 10)
	if err != nil {
		t.Fatalf("Autocomplete: %v", err)
	}
	if len(suggestions) == 0 {
		t.Error("Autocomplete returned no suggestions")
	}
}

func TestGetSections(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	sections, err := db.GetSections()
	if err != nil {
		t.Fatalf("GetSections: %v", err)
	}
	if len(sections) == 0 {
		t.Error("GetSections returned nothing")
	}
}

func TestGetPlatforms(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	platforms, err := db.GetPlatforms()
	if err != nil {
		t.Fatalf("GetPlatforms: %v", err)
	}
	if len(platforms) == 0 {
		t.Error("GetPlatforms returned nothing")
	}
}

func TestGetStats(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalPages == 0 {
		t.Errorf("GetStats.TotalPages = 0, want > 0")
	}
}

func TestGetPopular(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	popular, err := db.GetPopular(10)
	if err != nil {
		t.Fatalf("GetPopular: %v", err)
	}
	_ = popular
}

func TestRecordView(t *testing.T) {
	db := newTestDB(t)
	page := samplePage("ls", "1", "linux")
	if err := db.InsertManPage(page); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}
	if err := db.RecordView(page.ID); err != nil {
		t.Fatalf("RecordView: %v", err)
	}
}

func TestWhatis(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	results, err := db.Whatis("ls")
	if err != nil {
		t.Fatalf("Whatis: %v", err)
	}
	if len(results) == 0 {
		t.Error("Whatis returned no results")
	}
}

func TestApropos(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	results, err := db.Apropos("directory")
	if err != nil {
		t.Fatalf("Apropos: %v", err)
	}
	if len(results) == 0 {
		t.Error("Apropos returned no results")
	}
}

func TestBrowse(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	pages, total, err := db.Browse("", "", 1, 50)
	if err != nil {
		t.Fatalf("Browse: %v", err)
	}
	if total == 0 || len(pages) == 0 {
		t.Errorf("Browse = total=%d pages=%+v, want non-empty", total, pages)
	}
}

func TestBrowse_FilteredBySection(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	pages, total, err := db.Browse("1", "", 1, 50)
	if err != nil {
		t.Fatalf("Browse: %v", err)
	}
	if total == 0 || len(pages) == 0 {
		t.Errorf("Browse(section=1) = total=%d pages=%+v, want non-empty", total, pages)
	}
}

func TestCompare(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage linux: %v", err)
	}
	if err := db.InsertManPage(samplePage("ls", "1", "macos")); err != nil {
		t.Fatalf("InsertManPage macos: %v", err)
	}

	result, err := db.Compare("ls", "1")
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if result == nil {
		t.Fatal("Compare returned nil result")
	}
}

func TestGetTLDR(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	tldr, err := db.GetTLDR("ls", "1")
	if err != nil {
		t.Fatalf("GetTLDR: %v", err)
	}
	if tldr == nil {
		t.Fatal("GetTLDR returned nil")
	}
}

func TestGetRecentPages(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	entries, err := db.GetRecentPages("", "", 50)
	if err != nil {
		t.Fatalf("GetRecentPages: %v", err)
	}
	if len(entries) == 0 {
		t.Error("GetRecentPages returned nothing")
	}
}

func TestGetAllPageURLs(t *testing.T) {
	db := newTestDB(t)
	if err := db.InsertManPage(samplePage("ls", "1", "linux")); err != nil {
		t.Fatalf("InsertManPage: %v", err)
	}

	urls, err := db.GetAllPageURLs()
	if err != nil {
		t.Fatalf("GetAllPageURLs: %v", err)
	}
	if len(urls) == 0 {
		t.Error("GetAllPageURLs returned nothing")
	}
}

func TestLogAndGetAuditLogs(t *testing.T) {
	db := newTestDB(t)
	event := AuditEvent{
		Level:      "info",
		Category:   "system",
		Action:     "test_action",
		ActorType:  "system",
		ActorID:    "tester",
		ActorIP:    "127.0.0.1",
		TargetType: "test",
		TargetID:   "1",
		Details:    `{"key":"value"}`,
		Success:    true,
	}
	if err := db.LogAudit(event); err != nil {
		t.Fatalf("LogAudit: %v", err)
	}

	logs, err := db.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("GetAuditLogs: %v", err)
	}
	if len(logs) == 0 {
		t.Error("GetAuditLogs returned nothing")
	}
}
