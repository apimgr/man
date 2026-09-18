package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/casapps/casman/src/client/api"
)

func newTestModel() model {
	ti := textinput.New()
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 80, 20)
	vp := viewport.New(80, 20)
	return model{
		client:      api.New("http://example.com", ""),
		serverURL:   "http://example.com",
		state:       viewSearch,
		searchInput: ti,
		resultsList: l,
		viewport:    vp,
	}
}

func TestResultItem(t *testing.T) {
	i := resultItem{name: "ls", section: "1", title: "list files"}
	if i.Title() != "ls(1)" {
		t.Errorf("Title() = %q, want ls(1)", i.Title())
	}
	if i.Description() != "list files" {
		t.Errorf("Description() = %q, want list files", i.Description())
	}
	if i.FilterValue() != "ls" {
		t.Errorf("FilterValue() = %q, want ls", i.FilterValue())
	}
}

func TestStripBasicHTML(t *testing.T) {
	cases := map[string]string{
		"<p>Hello <b>world</b></p>":         "Hello world",
		"no tags":                           "no tags",
		"a &amp; b &lt;c&gt; &quot;q&quot;": "a & b <c> \"q\"",
		"":                                  "",
		"<unclosed":                         "<unclosed",
	}
	for in, want := range cases {
		if got := stripBasicHTML(in); got != want {
			t.Errorf("stripBasicHTML(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModelInit(t *testing.T) {
	m := newTestModel()
	if m.Init() == nil {
		t.Error("Init() returned nil cmd")
	}
}

func TestFormatManPage(t *testing.T) {
	m := newTestModel()

	page := &api.ManPage{
		Name: "ls", Section: "1", Title: "list files",
		Platform:    "linux",
		ContentText: "usage: ls",
		SeeAlso: []api.SeeAlsoEntry{
			{Name: "dir", Section: "1"},
			{Name: "vdir"},
		},
	}
	out := m.formatManPage(page)
	if !strings.Contains(out, "usage: ls") {
		t.Errorf("missing content: %q", out)
	}
	if !strings.Contains(out, "Platform: linux") {
		t.Errorf("missing platform: %q", out)
	}
	if !strings.Contains(out, "dir(1)") || !strings.Contains(out, "vdir") {
		t.Errorf("missing see-also entries: %q", out)
	}
}

func TestFormatManPage_MarkdownFallback(t *testing.T) {
	m := newTestModel()
	page := &api.ManPage{Name: "ls", Section: "1", ContentMarkdown: "# usage"}
	out := m.formatManPage(page)
	if !strings.Contains(out, "# usage") {
		t.Errorf("missing markdown content: %q", out)
	}
}

func TestFormatManPage_HTMLFallback(t *testing.T) {
	m := newTestModel()
	page := &api.ManPage{Name: "ls", Section: "1", ContentHTML: "<p>usage</p>"}
	out := m.formatManPage(page)
	if !strings.Contains(out, "usage") {
		t.Errorf("missing stripped html content: %q", out)
	}
}

func TestViewSearch(t *testing.T) {
	m := newTestModel()
	out := m.viewSearch()
	if !strings.Contains(out, "Enter a command name") {
		t.Errorf("unexpected viewSearch: %q", out)
	}

	m.searching = true
	out = m.viewSearch()
	if !strings.Contains(out, "Searching...") {
		t.Errorf("unexpected viewSearch while searching: %q", out)
	}
}

func TestViewResults_Empty(t *testing.T) {
	m := newTestModel()
	out := m.viewResults()
	if !strings.Contains(out, "No results found") {
		t.Errorf("unexpected viewResults: %q", out)
	}
}

func TestViewResults_WithItems(t *testing.T) {
	m := newTestModel()
	m.results = []api.SearchResult{{Name: "ls", Section: "1"}}
	out := m.viewResults()
	if out == "" {
		t.Error("viewResults returned empty string with items present")
	}
}

func TestViewManPage_Loading(t *testing.T) {
	m := newTestModel()
	out := m.viewManPage()
	if !strings.Contains(out, "Loading...") {
		t.Errorf("unexpected viewManPage: %q", out)
	}
}

func TestViewManPage_WithPage(t *testing.T) {
	m := newTestModel()
	m.manPage = &api.ManPage{Name: "ls"}
	m.viewport.SetContent("usage: ls")
	out := m.viewManPage()
	if !strings.Contains(out, "usage: ls") {
		t.Errorf("unexpected viewManPage: %q", out)
	}
}

func TestViewHelp(t *testing.T) {
	m := newTestModel()
	cases := []struct {
		state viewState
		want  string
	}{
		{viewSearch, "enter: search"},
		{viewResults, "navigate"},
		{viewManPage, "scroll"},
	}
	for _, c := range cases {
		m.state = c.state
		if got := m.viewHelp(); !strings.Contains(got, c.want) {
			t.Errorf("viewHelp(state=%v) = %q, want to contain %q", c.state, got, c.want)
		}
	}
}

func TestView_Quitting(t *testing.T) {
	m := newTestModel()
	m.quitting = true
	if got := m.View(); got != "" {
		t.Errorf("View() while quitting = %q, want empty", got)
	}
}

func TestView_WithError(t *testing.T) {
	m := newTestModel()
	m.err = errTest("boom")
	out := m.View()
	if !strings.Contains(out, "boom") {
		t.Errorf("View() missing error text: %q", out)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestUpdate_WindowSize(t *testing.T) {
	m := newTestModel()
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	mm := newM.(model)
	if mm.width != 100 || mm.height != 40 {
		t.Errorf("width/height = %d/%d, want 100/40", mm.width, mm.height)
	}
}

func TestUpdate_QuitFromSearch(t *testing.T) {
	m := newTestModel()
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	mm := newM.(model)
	if !mm.quitting {
		t.Error("expected quitting = true")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestUpdate_QBackFromResults(t *testing.T) {
	m := newTestModel()
	m.state = viewResults
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	mm := newM.(model)
	if mm.state != viewSearch {
		t.Errorf("state = %v, want viewSearch", mm.state)
	}
}

func TestUpdate_EscFromResults(t *testing.T) {
	m := newTestModel()
	m.state = viewResults
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newM.(model)
	if mm.state != viewSearch {
		t.Errorf("state = %v, want viewSearch", mm.state)
	}
}

func TestUpdate_EscFromManPage(t *testing.T) {
	m := newTestModel()
	m.state = viewManPage
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newM.(model)
	if mm.state != viewResults {
		t.Errorf("state = %v, want viewResults", mm.state)
	}
}

func TestUpdate_EnterSearchEmptyQuery(t *testing.T) {
	m := newTestModel()
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := newM.(model)
	if mm.searching {
		t.Error("should not start searching on empty query")
	}
	_ = cmd
}

func TestUpdate_EnterSearchWithQuery(t *testing.T) {
	m := newTestModel()
	m.searchInput.SetValue("ls")
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := newM.(model)
	if !mm.searching {
		t.Error("expected searching = true")
	}
	if cmd == nil {
		t.Error("expected doSearch cmd")
	}
}

func TestUpdate_SearchResultMsg_Error(t *testing.T) {
	m := newTestModel()
	newM, _ := m.Update(searchResultMsg{err: errTest("failed")})
	mm := newM.(model)
	if mm.err == nil {
		t.Error("expected err to be set")
	}
	if mm.searching {
		t.Error("searching should be false after result")
	}
}

func TestUpdate_SearchResultMsg_Success(t *testing.T) {
	m := newTestModel()
	newM, _ := m.Update(searchResultMsg{results: []api.SearchResult{{Name: "ls", Section: "1"}}})
	mm := newM.(model)
	if mm.state != viewResults {
		t.Errorf("state = %v, want viewResults", mm.state)
	}
	if len(mm.results) != 1 {
		t.Errorf("results len = %d, want 1", len(mm.results))
	}
}

func TestUpdate_ManPageMsg_Error(t *testing.T) {
	m := newTestModel()
	newM, _ := m.Update(manPageMsg{err: errTest("failed")})
	mm := newM.(model)
	if mm.err == nil {
		t.Error("expected err to be set")
	}
}

func TestUpdate_ManPageMsg_Success(t *testing.T) {
	m := newTestModel()
	newM, _ := m.Update(manPageMsg{page: &api.ManPage{Name: "ls", ContentText: "usage: ls"}})
	mm := newM.(model)
	if mm.state != viewManPage {
		t.Errorf("state = %v, want viewManPage", mm.state)
	}
	if mm.manPage == nil {
		t.Fatal("manPage not set")
	}
}

func TestDoSearch(t *testing.T) {
	m := newTestModel()
	cmd := m.doSearch("ls")
	if cmd == nil {
		t.Fatal("doSearch returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(searchResultMsg); !ok {
		t.Errorf("cmd() returned %T, want searchResultMsg", msg)
	}
}

func TestFetchManPage(t *testing.T) {
	m := newTestModel()
	cmd := m.fetchManPage("ls", "1")
	if cmd == nil {
		t.Fatal("fetchManPage returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(manPageMsg); !ok {
		t.Errorf("cmd() returned %T, want manPageMsg", msg)
	}
}
