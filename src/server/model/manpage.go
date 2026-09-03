// Package model defines data structures for casman.
package model

import "time"

// ManPage represents a single man page entry.
type ManPage struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Section  string `json:"section"`
	Title    string `json:"title"`
	Platform string `json:"platform"`
	Distro   string `json:"distro,omitempty"`
	Version  string `json:"version,omitempty"`
	Language string `json:"language,omitempty"`

	// Source information
	SourceFormat string `json:"source_format"` // groff, mdoc, markdown
	SourceRaw    string `json:"-"`             // Original source (not in JSON)
	SourceURL    string `json:"source_url,omitempty"`

	// Pre-rendered content
	ContentHTML     string `json:"content_html,omitempty"`
	ContentText     string `json:"content_text,omitempty"`
	ContentMarkdown string `json:"content_markdown,omitempty"`
	ContentRaw      string `json:"content_raw,omitempty"`

	// Metadata
	Synopsis    string         `json:"synopsis,omitempty"`
	Description string         `json:"description,omitempty"`
	SeeAlso     []SeeAlsoEntry `json:"see_also,omitempty"`
	SearchText  string         `json:"-"` // Normalized searchable text

	// Timestamps
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ManPageSummary is a lightweight man page representation for listings.
type ManPageSummary struct {
	Name     string `json:"name"`
	Section  string `json:"section"`
	Title    string `json:"title"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// SeeAlsoEntry represents a reference to another man page.
type SeeAlsoEntry struct {
	Name    string `json:"name"`
	Section string `json:"section"`
	URL     string `json:"url"`
}

// Section represents a man page section.
type Section struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

// Platform represents an OS/platform.
type Platform struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	Name     string  `json:"name"`
	Section  string  `json:"section"`
	Title    string  `json:"title"`
	Platform string  `json:"platform"`
	Distro   string  `json:"distro,omitempty"`
	Snippet  string  `json:"snippet,omitempty"`
	Score    float64 `json:"score"`
	URL      string  `json:"url"`
}

// SearchResponse represents the full search response.
type SearchResponse struct {
	Query       string          `json:"query"`
	Total       int             `json:"total"`
	Page        int             `json:"page"`
	Limit       int             `json:"limit"`
	Results     []SearchResult  `json:"results"`
	Suggestions []string        `json:"suggestions,omitempty"`
	Filters     *SearchFilters  `json:"filters,omitempty"`
}

// SearchFilters contains available filter options with counts.
type SearchFilters struct {
	Sections  []FilterOption `json:"sections"`
	Platforms []FilterOption `json:"platforms"`
}

// FilterOption represents a filter option with count.
type FilterOption struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// TLDR represents a quick summary for a man page.
type TLDR struct {
	Name           string        `json:"name"`
	Section        string        `json:"section"`
	OneLiner       string        `json:"one_liner"`
	CommonExamples []TLDRExample `json:"common_examples"`
	KeyOptions     []TLDROption  `json:"key_options"`
	GeneratedAt    time.Time     `json:"generated_at"`
	Source         string        `json:"source"` // auto or manual
}

// TLDRExample represents a common usage example.
type TLDRExample struct {
	Command     string `json:"cmd"`
	Description string `json:"desc"`
}

// TLDROption represents a key option.
type TLDROption struct {
	Flag        string `json:"flag"`
	Description string `json:"desc"`
}

// Stats represents database statistics.
type Stats struct {
	TotalPages     int            `json:"total_pages"`
	TotalSections  int            `json:"total_sections"`
	TotalPlatforms int            `json:"total_platforms"`
	TotalLanguages int            `json:"total_languages"`
	BySection      map[string]int `json:"by_section"`
	ByPlatform     map[string]int `json:"by_platform"`
	LastUpdated    time.Time      `json:"last_updated"`
}

// CompareResult represents a comparison of a man page across platforms.
type CompareResult struct {
	Name      string              `json:"name"`
	Section   string              `json:"section"`
	Platforms []ComparePlatform   `json:"platforms"`
}

// ComparePlatform represents a single platform's version in a comparison.
type ComparePlatform struct {
	Platform    string   `json:"platform"`
	Title       string   `json:"title"`
	Synopsis    string   `json:"synopsis"`
	Options     []string `json:"options,omitempty"`
	ContentHTML string   `json:"content_html,omitempty"`
	Available   bool     `json:"available"`
}

// WhatisResult represents a whatis lookup result.
type WhatisResult struct {
	Name     string `json:"name"`
	Section  string `json:"section"`
	Title    string `json:"title"`
	Platform string `json:"platform"`
}

// PopularPage represents a popular/trending man page.
type PopularPage struct {
	Name     string `json:"name"`
	Section  string `json:"section"`
	Title    string `json:"title"`
	Platform string `json:"platform"`
	Views    int    `json:"views"`
}

// FeedEntry represents an entry in RSS/Atom feed.
type FeedEntry struct {
	Name      string    `json:"name"`
	Section   string    `json:"section"`
	Title     string    `json:"title"`
	Platform  string    `json:"platform"`
	Summary   string    `json:"summary"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at"`
	IsNew     bool      `json:"is_new"`
}

// Sections defines all standard man page sections.
var Sections = []Section{
	{ID: "1", Name: "User Commands", Description: "Executable programs, shell commands"},
	{ID: "2", Name: "System Calls", Description: "Kernel system calls"},
	{ID: "3", Name: "Library Functions", Description: "C library functions"},
	{ID: "4", Name: "Devices", Description: "Device files, special files (/dev/*)"},
	{ID: "5", Name: "File Formats", Description: "Configuration file formats"},
	{ID: "6", Name: "Games", Description: "Games and screensavers"},
	{ID: "7", Name: "Miscellaneous", Description: "Conventions, protocols, standards"},
	{ID: "8", Name: "Admin Commands", Description: "System administration commands"},
	{ID: "9", Name: "Kernel", Description: "Kernel routines (Linux-specific)"},
	{ID: "n", Name: "New/Tcl", Description: "New documentation, Tcl/Tk commands"},
	{ID: "x", Name: "X Window System", Description: "X11/Xorg related documentation"},
}

// Platforms defines all supported platforms.
var Platforms = []Platform{
	{ID: "linux", Name: "Linux", Description: "Linux distributions"},
	{ID: "freebsd", Name: "FreeBSD", Description: "FreeBSD operating system"},
	{ID: "openbsd", Name: "OpenBSD", Description: "OpenBSD operating system"},
	{ID: "netbsd", Name: "NetBSD", Description: "NetBSD operating system"},
	{ID: "dragonfly", Name: "DragonFly BSD", Description: "DragonFly BSD operating system"},
	{ID: "macos", Name: "macOS", Description: "Apple macOS / Darwin"},
	{ID: "void", Name: "Void Linux", Description: "Void Linux distribution"},
	{ID: "alpine", Name: "Alpine Linux", Description: "Alpine Linux distribution"},
	{ID: "posix", Name: "POSIX", Description: "POSIX standard"},
}

// HealthResponse represents the /healthz response per PART 13.
type HealthResponse struct {
	// Project identification
	Project ProjectInfo `json:"project"`

	// Overall status
	Status         string   `json:"status"`
	PendingRestart bool     `json:"pending_restart,omitempty"`
	RestartReason  []string `json:"restart_reason,omitempty"`

	// Version & build info
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	Build     BuildInfo `json:"build"`

	// Runtime info
	Uptime    string    `json:"uptime"`
	Mode      string    `json:"mode"`
	Timestamp time.Time `json:"timestamp"`

	// Cluster info
	Cluster ClusterInfo `json:"cluster"`

	// Features
	Features FeaturesInfo `json:"features"`

	// Component health checks
	Checks ChecksInfo `json:"checks"`

	// Statistics
	Stats HealthStats `json:"stats"`

	// App-specific data
	AppData AppDataInfo `json:"app_data,omitempty"`
}

// ProjectInfo contains branding information.
type ProjectInfo struct {
	Name        string `json:"name"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
}

// BuildInfo contains build-time information.
type BuildInfo struct {
	Commit string `json:"commit"`
	Date   string `json:"date"`
}

// ClusterInfo contains cluster status.
type ClusterInfo struct {
	Enabled   bool     `json:"enabled"`
	Status    string   `json:"status,omitempty"`
	Primary   string   `json:"primary,omitempty"`
	Nodes     []string `json:"nodes,omitempty"`
	NodeCount int      `json:"node_count,omitempty"`
	Role      string   `json:"role,omitempty"`
}

// FeaturesInfo contains enabled features.
type FeaturesInfo struct {
	Tor   TorInfo `json:"tor"`
	GeoIP bool    `json:"geoip"`
}

// TorInfo contains Tor hidden service status.
type TorInfo struct {
	Enabled  bool   `json:"enabled"`
	Running  bool   `json:"running"`
	Status   string `json:"status"`
	Hostname string `json:"hostname,omitempty"`
}

// ChecksInfo contains component health status.
type ChecksInfo struct {
	Database  string `json:"database"`
	Cache     string `json:"cache"`
	Disk      string `json:"disk"`
	Scheduler string `json:"scheduler"`
	Cluster   string `json:"cluster,omitempty"`
	Tor       string `json:"tor,omitempty"`
}

// HealthStats contains public-safe statistics.
type HealthStats struct {
	RequestsTotal int64 `json:"requests_total"`
	Requests24h   int64 `json:"requests_24h"`
	ActiveConns   int   `json:"active_connections"`
	// casman-specific
	ManPagesTotal int `json:"man_pages_total"`
	PlatformCount int `json:"platform_count"`
	SectionCount  int `json:"section_count"`
}

// AppDataInfo contains casman-specific data.
type AppDataInfo struct {
	ManPages   int `json:"man_pages"`
	Platforms  int `json:"platforms"`
	Sections   int `json:"sections"`
	SearchHits int `json:"search_hits_24h"`
}
