// Package template provides HTML templates for casman.
package template

import (
	"embed"
	htmltemplate "html/template"
	"io"
	"strings"

	"github.com/casapps/casman/src/server/model"
)

//go:embed *.html
var templateFS embed.FS

// Templates holds all parsed templates.
type Templates struct {
	templates *htmltemplate.Template
}

// New creates a new Templates instance.
func New() (*Templates, error) {
	funcMap := htmltemplate.FuncMap{
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"title":    strings.Title,
		"contains": strings.Contains,
		"join":     strings.Join,
		"safe":     func(s string) htmltemplate.HTML { return htmltemplate.HTML(s) },
		"add":      func(a, b int) int { return a + b },
		"sub":      func(a, b int) int { return a - b },
	}

	t, err := htmltemplate.New("").Funcs(funcMap).ParseFS(templateFS, "*.html")
	if err != nil {
		return nil, err
	}

	return &Templates{templates: t}, nil
}

// Render renders a template to the writer.
// It first clones the templates, then parses the specific page template to define "content",
// and finally executes the "base" template.
func (t *Templates) Render(w io.Writer, name string, data interface{}) error {
	// Clone the base templates
	tmpl, err := t.templates.Clone()
	if err != nil {
		return err
	}

	// Parse the specific page template to define "content"
	pageContent, err := templateFS.ReadFile(name)
	if err != nil {
		return err
	}

	_, err = tmpl.Parse(string(pageContent))
	if err != nil {
		return err
	}

	// Execute the base template which includes the content
	return tmpl.ExecuteTemplate(w, "base", data)
}

// HomeData holds data for the home page.
type HomeData struct {
	Title       string
	Tagline     string
	Stats       model.Stats
	Popular     []model.ManPageSummary
	Sections    []model.Section
	Platforms   []model.Platform
	RecentPages []model.ManPageSummary
	Version     string
}

// ManPageData holds data for man page view.
type ManPageData struct {
	Title          string
	Name           string
	Section        string
	PageTitle      string
	Platform       string
	Synopsis       string
	ContentHTML    htmltemplate.HTML
	SeeAlso        []model.SeeAlsoEntry
	OtherPlatforms []string
	Bookmarked     bool
	Version        string
	// TLDR is the auto-generated quick summary; nil when unavailable so the
	// template can omit the card entirely. Per IDEA.md "TLDR Display".
	TLDR *model.TLDR
}

// SearchData holds data for search results.
type SearchData struct {
	Title     string
	Query     string
	Section   string
	Platform  string
	Results   []model.SearchResult
	Total     int
	Page      int
	Limit     int
	HasMore   bool
	Sections  []model.Section
	Platforms []model.Platform
	Version   string
}

// BrowseData holds data for browse page.
type BrowseData struct {
	Title     string
	Section   string
	Platform  string
	Pages     []model.ManPageSummary
	Total     int
	Page      int
	Limit     int
	HasMore   bool
	Sections  []model.Section
	Platforms []model.Platform
	Version   string
}

// CompareData holds data for compare page.
type CompareData struct {
	Title   string
	Name    string
	Section string
	Result  *model.CompareResult
	Version string
}

// ErrorData holds data for error pages.
type ErrorData struct {
	Title   string
	Code    int
	Message string
	Version string
}

// HealthData holds data for health page.
type HealthData struct {
	Title   string
	Health  model.HealthResponse
	Version string
}
