// Package tui provides terminal user interface components using bubbletea.
// Per PART 33: Client TUI must be professional and fully functional.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/casapps/casman/src/client/api"
)

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

// View states
type viewState int

const (
	viewSearch viewState = iota
	viewResults
	viewManPage
)

// model is the main TUI model
type model struct {
	client    *api.Client
	width     int
	height    int
	state     viewState
	err       error
	quitting  bool
	serverURL string

	// Search components
	searchInput textinput.Model
	searching   bool

	// Results components
	resultsList list.Model
	results     []api.SearchResult

	// Man page viewer
	viewport viewport.Model
	manPage  *api.ManPage
}

// resultItem implements list.Item for search results
type resultItem struct {
	name     string
	section  string
	title    string
	platform string
}

func (i resultItem) Title() string       { return fmt.Sprintf("%s(%s)", i.name, i.section) }
func (i resultItem) Description() string { return i.title }
func (i resultItem) FilterValue() string { return i.name }

// Messages
type searchResultMsg struct {
	results []api.SearchResult
	err     error
}

type manPageMsg struct {
	page *api.ManPage
	err  error
}

// Run starts the TUI application.
func Run(serverURL, token string) error {
	client := api.New(serverURL, token)

	// Create search input
	ti := textinput.New()
	ti.Placeholder = "Search man pages..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// Create results list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedStyle
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Search Results"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)

	// Create viewport
	vp := viewport.New(80, 20)

	m := model{
		client:      client,
		serverURL:   serverURL,
		state:       viewSearch,
		searchInput: ti,
		resultsList: l,
		viewport:    vp,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handlers
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == viewSearch {
				m.quitting = true
				return m, tea.Quit
			}
			// Go back to search
			m.state = viewSearch
			m.err = nil
			return m, nil

		case "esc":
			switch m.state {
			case viewResults:
				m.state = viewSearch
				m.searchInput.Focus()
			case viewManPage:
				m.state = viewResults
			}
			return m, nil

		case "enter":
			switch m.state {
			case viewSearch:
				if m.searchInput.Value() != "" {
					m.searching = true
					return m, m.doSearch(m.searchInput.Value())
				}
			case viewResults:
				if len(m.results) > 0 {
					selected := m.resultsList.SelectedItem()
					if item, ok := selected.(resultItem); ok {
						return m, m.fetchManPage(item.name, item.section)
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resultsList.SetSize(msg.Width-4, msg.Height-8)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6

	case searchResultMsg:
		m.searching = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.results = msg.results
		m.state = viewResults

		// Convert to list items
		items := make([]list.Item, len(msg.results))
		for i, r := range msg.results {
			items[i] = resultItem{
				name:     r.Name,
				section:  r.Section,
				title:    r.Title,
				platform: r.Platform,
			}
		}
		m.resultsList.SetItems(items)
		return m, nil

	case manPageMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.manPage = msg.page
		m.state = viewManPage

		// Set viewport content
		content := m.formatManPage(msg.page)
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
		return m, nil
	}

	// Update current view component
	switch m.state {
	case viewSearch:
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	case viewResults:
		m.resultsList, cmd = m.resultsList.Update(msg)
		cmds = append(cmds, cmd)
	case viewManPage:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Header
	header := titleStyle.Render(" CASMAN - Universal Man Pages ")
	s.WriteString(header)
	s.WriteString("\n")
	s.WriteString(infoStyle.Render(fmt.Sprintf(" Server: %s", m.serverURL)))
	s.WriteString("\n\n")

	// Error display
	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\n")
	}

	// Main content
	switch m.state {
	case viewSearch:
		s.WriteString(m.viewSearch())
	case viewResults:
		s.WriteString(m.viewResults())
	case viewManPage:
		s.WriteString(m.viewManPage())
	}

	// Help
	s.WriteString("\n")
	s.WriteString(m.viewHelp())

	return s.String()
}

func (m model) viewSearch() string {
	var s strings.Builder

	s.WriteString("Search for man pages:\n\n")
	s.WriteString(borderStyle.Render(m.searchInput.View()))
	s.WriteString("\n\n")

	if m.searching {
		s.WriteString(infoStyle.Render("Searching..."))
	} else {
		s.WriteString(infoStyle.Render("Enter a command name or keyword to search"))
	}

	return s.String()
}

func (m model) viewResults() string {
	if len(m.results) == 0 {
		return infoStyle.Render("No results found. Press Esc to search again.")
	}

	return m.resultsList.View()
}

func (m model) viewManPage() string {
	if m.manPage == nil {
		return infoStyle.Render("Loading...")
	}

	return m.viewport.View()
}

func (m model) viewHelp() string {
	var help string
	switch m.state {
	case viewSearch:
		help = "enter: search | q/ctrl+c: quit"
	case viewResults:
		help = "↑/↓: navigate | enter: view | esc: back | q: quit"
	case viewManPage:
		help = "↑/↓/pgup/pgdn: scroll | esc: back | q: quit"
	}
	return helpStyle.Render(help)
}

// formatManPage formats a man page for display
func (m model) formatManPage(page *api.ManPage) string {
	var s strings.Builder

	// Header
	header := fmt.Sprintf("%s(%s) - %s", page.Name, page.Section, page.Title)
	s.WriteString(titleStyle.Render(header))
	s.WriteString("\n\n")

	// Platform info
	if page.Platform != "" {
		s.WriteString(infoStyle.Render(fmt.Sprintf("Platform: %s", page.Platform)))
		s.WriteString("\n\n")
	}

	// Content - prefer text, fall back to markdown
	content := page.ContentText
	if content == "" {
		content = page.ContentMarkdown
	}
	if content == "" {
		// Strip HTML as last resort
		content = stripBasicHTML(page.ContentHTML)
	}

	s.WriteString(content)

	// See also
	if len(page.SeeAlso) > 0 {
		s.WriteString("\n\nSEE ALSO\n")
		for _, sa := range page.SeeAlso {
			if sa.Section != "" {
				s.WriteString(fmt.Sprintf("    %s(%s)\n", sa.Name, sa.Section))
			} else {
				s.WriteString(fmt.Sprintf("    %s\n", sa.Name))
			}
		}
	}

	return s.String()
}

// doSearch performs a search query
func (m model) doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Search(query, "", "", 1)
		if err != nil {
			return searchResultMsg{err: err}
		}
		return searchResultMsg{results: resp.Results}
	}
}

// fetchManPage retrieves a man page
func (m model) fetchManPage(name, section string) tea.Cmd {
	return func() tea.Msg {
		var page *api.ManPage
		var err error

		if section != "" {
			page, err = m.client.GetManPageSection(section, name)
		} else {
			page, err = m.client.GetManPage(name)
		}

		if err != nil {
			return manPageMsg{err: err}
		}
		return manPageMsg{page: page}
	}
}

// stripBasicHTML removes basic HTML tags
func stripBasicHTML(html string) string {
	// Simple tag removal
	result := html
	for strings.Contains(result, "<") {
		start := strings.Index(result, "<")
		end := strings.Index(result, ">")
		if start >= 0 && end > start {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}

	// Common entities
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")

	return result
}
