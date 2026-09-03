// Package cmd implements CLI commands for the casman client.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/casapps/casman/src/client/api"
)

// Config holds the CLI configuration.
type Config struct {
	ServerURL string
	Token     string
	Format    string
	Pager     bool
}

// Runner handles CLI commands.
type Runner struct {
	client *api.Client
	config *Config
}

// New creates a new command runner.
func New(cfg *Config) *Runner {
	return &Runner{
		client: api.New(cfg.ServerURL, cfg.Token),
		config: cfg,
	}
}

// Man displays a man page.
func (r *Runner) Man(name string, section string) error {
	var page *api.ManPage
	var err error

	if section != "" {
		page, err = r.client.GetManPageSection(section, name)
	} else {
		page, err = r.client.GetManPage(name)
	}

	if err != nil {
		return fmt.Errorf("fetching man page: %w", err)
	}

	// Choose content format
	content := page.ContentText
	if content == "" {
		// Fall back to cleaning HTML
		content = stripHTML(page.ContentHTML)
	}
	if content == "" {
		content = page.ContentMarkdown
	}
	if content == "" {
		return fmt.Errorf("no content available for %s", name)
	}

	// Format header
	header := fmt.Sprintf("%s(%s) - %s\n\n", page.Name, page.Section, page.Title)

	// Display with pager if enabled
	if r.config.Pager {
		return r.displayWithPager(header + content)
	}

	fmt.Print(header + content)
	return nil
}

// Search searches for man pages.
func (r *Runner) Search(query string, section, platform string) error {
	resp, err := r.client.Search(query, section, platform, 1)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	if len(resp.Results) == 0 {
		fmt.Printf("No results found for '%s'\n", query)
		return nil
	}

	fmt.Printf("Found %d results for '%s':\n\n", resp.Total, query)

	for _, result := range resp.Results {
		fmt.Printf("  %s(%s) - %s\n", result.Name, result.Section, result.Title)
		if result.Snippet != "" {
			// Clean up snippet
			snippet := strings.ReplaceAll(result.Snippet, "<mark>", "")
			snippet = strings.ReplaceAll(snippet, "</mark>", "")
			fmt.Printf("    %s\n", snippet)
		}
		fmt.Println()
	}

	if resp.Total > len(resp.Results) {
		fmt.Printf("... and %d more results\n", resp.Total-len(resp.Results))
	}

	return nil
}

// Whatis displays one-line descriptions.
func (r *Runner) Whatis(name string) error {
	results, err := r.client.Whatis(name)
	if err != nil {
		return fmt.Errorf("whatis: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("%s: nothing appropriate\n", name)
		return nil
	}

	for _, result := range results {
		fmt.Printf("%s(%s) - %s\n", result.Name, result.Section, result.Title)
	}

	return nil
}

// Apropos searches descriptions.
func (r *Runner) Apropos(query string) error {
	results, err := r.client.Apropos(query)
	if err != nil {
		return fmt.Errorf("apropos: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("%s: nothing appropriate\n", query)
		return nil
	}

	for _, result := range results {
		fmt.Printf("%s(%s) - %s\n", result.Name, result.Section, result.Title)
	}

	return nil
}

// Stats displays database statistics.
func (r *Runner) Stats() error {
	stats, err := r.client.GetStats()
	if err != nil {
		return fmt.Errorf("fetching stats: %w", err)
	}

	fmt.Println("Database Statistics:")
	fmt.Printf("  Total Pages:     %d\n", stats.TotalPages)
	fmt.Printf("  Total Sections:  %d\n", stats.TotalSections)
	fmt.Printf("  Total Platforms: %d\n", stats.TotalPlatforms)
	fmt.Printf("  Total Languages: %d\n", stats.TotalLanguages)

	if len(stats.BySection) > 0 {
		fmt.Println("\n  By Section:")
		for section, count := range stats.BySection {
			fmt.Printf("    Section %s: %d pages\n", section, count)
		}
	}

	if len(stats.ByPlatform) > 0 {
		fmt.Println("\n  By Platform:")
		for platform, count := range stats.ByPlatform {
			fmt.Printf("    %s: %d pages\n", platform, count)
		}
	}

	return nil
}

// Sections lists all sections.
func (r *Runner) Sections() error {
	sections, err := r.client.GetSections()
	if err != nil {
		return fmt.Errorf("fetching sections: %w", err)
	}

	fmt.Println("Man Page Sections:")
	for _, s := range sections {
		if s.Count > 0 {
			fmt.Printf("  %s  %-30s (%d pages)\n", s.ID, s.Name, s.Count)
		} else {
			fmt.Printf("  %s  %s\n", s.ID, s.Name)
		}
	}

	return nil
}

// Platforms lists all platforms.
func (r *Runner) Platforms() error {
	platforms, err := r.client.GetPlatforms()
	if err != nil {
		return fmt.Errorf("fetching platforms: %w", err)
	}

	fmt.Println("Supported Platforms:")
	for _, p := range platforms {
		if p.Count > 0 {
			fmt.Printf("  %-12s %s (%d pages)\n", p.ID, p.Name, p.Count)
		} else {
			fmt.Printf("  %-12s %s\n", p.ID, p.Name)
		}
	}

	return nil
}

// HealthCheck checks server health.
func (r *Runner) HealthCheck() error {
	healthy, err := r.client.HealthCheck()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if healthy {
		fmt.Println("Server is healthy")
	} else {
		fmt.Println("Server is not healthy")
	}

	return nil
}

// displayWithPager displays content using a pager (less/more).
func (r *Runner) displayWithPager(content string) error {
	// Try less first, then more
	pagers := []string{"less", "more"}

	for _, pager := range pagers {
		path, err := exec.LookPath(pager)
		if err != nil {
			continue
		}

		cmd := exec.Command(path)
		cmd.Stdin = strings.NewReader(content)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	// No pager available, just print
	fmt.Print(content)
	return nil
}

// stripHTML removes HTML tags from content.
func stripHTML(html string) string {
	// Remove script and style content (separate regexes - Go doesn't support backreferences)
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = reScript.ReplaceAllString(html, "")
	reStyle := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	// Remove HTML tags
	reTags := regexp.MustCompile(`<[^>]*>`)
	text := reTags.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	reSpace := regexp.MustCompile(`\s+`)
	text = reSpace.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
