// Package api provides the HTTP client for communicating with the casman server.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client represents an API client for the casman server.
type Client struct {
	ServerURL  string
	Token      string
	HTTPClient *http.Client
}

// ManPage represents a man page from the API.
type ManPage struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	Section         string         `json:"section"`
	Title           string         `json:"title"`
	Platform        string         `json:"platform"`
	Language        string         `json:"language"`
	SourceFormat    string         `json:"source_format,omitempty"`
	ContentHTML     string         `json:"content_html,omitempty"`
	ContentText     string         `json:"content_text,omitempty"`
	ContentMarkdown string         `json:"content_markdown,omitempty"`
	ContentRaw      string         `json:"content_raw,omitempty"`
	Synopsis        string         `json:"synopsis,omitempty"`
	Description     string         `json:"description,omitempty"`
	SeeAlso         []SeeAlsoEntry `json:"see_also,omitempty"`
}

// SeeAlsoEntry represents a reference to another man page.
type SeeAlsoEntry struct {
	Name    string `json:"name"`
	Section string `json:"section,omitempty"`
	URL     string `json:"url,omitempty"`
}

// SearchResult represents a search result.
type SearchResult struct {
	Name     string  `json:"name"`
	Section  string  `json:"section"`
	Title    string  `json:"title"`
	Platform string  `json:"platform"`
	Snippet  string  `json:"snippet,omitempty"`
	Score    float64 `json:"score,omitempty"`
	URL      string  `json:"url"`
}

// SearchResponse represents the API search response.
type SearchResponse struct {
	Query   string         `json:"query"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	Results []SearchResult `json:"results"`
}

// Stats represents database statistics.
type Stats struct {
	TotalPages     int            `json:"total_pages"`
	TotalSections  int            `json:"total_sections"`
	TotalPlatforms int            `json:"total_platforms"`
	TotalLanguages int            `json:"total_languages"`
	BySection      map[string]int `json:"by_section"`
	ByPlatform     map[string]int `json:"by_platform"`
}

// Section represents a man page section.
type Section struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

// Platform represents an OS platform.
type Platform struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// New creates a new API client.
func New(serverURL, token string) *Client {
	if serverURL == "" {
		serverURL = "http://localhost:64580"
	}

	return &Client{
		ServerURL: serverURL,
		Token:     token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetManPage retrieves a man page by name.
func (c *Client) GetManPage(name string) (*ManPage, error) {
	return c.getManPageFromURL(fmt.Sprintf("%s/api/v1/man/%s", c.ServerURL, url.PathEscape(name)))
}

// GetManPageSection retrieves a man page by section and name.
func (c *Client) GetManPageSection(section, name string) (*ManPage, error) {
	return c.getManPageFromURL(fmt.Sprintf("%s/api/v1/man/%s/%s", c.ServerURL, section, url.PathEscape(name)))
}

// GetManPagePlatform retrieves a man page by platform, section, and name.
func (c *Client) GetManPagePlatform(platform, section, name string) (*ManPage, error) {
	return c.getManPageFromURL(fmt.Sprintf("%s/api/v1/man/%s/%s/%s", c.ServerURL, platform, section, url.PathEscape(name)))
}

func (c *Client) getManPageFromURL(url string) (*ManPage, error) {
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var page ManPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &page, nil
}

// Search searches for man pages.
func (c *Client) Search(query string, section, platform string, page int) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	if section != "" {
		params.Set("section", section)
	}
	if platform != "" {
		params.Set("platform", platform)
	}
	if page > 1 {
		params.Set("page", fmt.Sprintf("%d", page))
	}

	url := fmt.Sprintf("%s/api/v1/search?%s", c.ServerURL, params.Encode())
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &resp, nil
}

// Whatis retrieves whatis information for a name.
func (c *Client) Whatis(name string) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/api/v1/whatis/%s", c.ServerURL, url.PathEscape(name))
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return results, nil
}

// Apropos searches descriptions for a keyword.
func (c *Client) Apropos(query string) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)

	url := fmt.Sprintf("%s/api/v1/apropos?%s", c.ServerURL, params.Encode())
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Query   string         `json:"query"`
		Results []SearchResult `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return resp.Results, nil
}

// GetStats retrieves database statistics.
func (c *Client) GetStats() (*Stats, error) {
	url := fmt.Sprintf("%s/api/v1/stats", c.ServerURL)
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var stats Stats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &stats, nil
}

// GetSections retrieves all sections.
func (c *Client) GetSections() ([]Section, error) {
	url := fmt.Sprintf("%s/api/v1/sections", c.ServerURL)
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var sections []Section
	if err := json.Unmarshal(body, &sections); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return sections, nil
}

// GetPlatforms retrieves all platforms.
func (c *Client) GetPlatforms() ([]Platform, error) {
	url := fmt.Sprintf("%s/api/v1/platforms", c.ServerURL)
	body, err := c.doRequest("GET", url)
	if err != nil {
		return nil, err
	}

	var platforms []Platform
	if err := json.Unmarshal(body, &platforms); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return platforms, nil
}

// HealthCheck checks if the server is healthy.
func (c *Client) HealthCheck() (bool, error) {
	url := fmt.Sprintf("%s/api/v1/healthz", c.ServerURL)
	body, err := c.doRequest("GET", url)
	if err != nil {
		return false, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return false, fmt.Errorf("parsing response: %w", err)
	}

	status, ok := resp["status"].(string)
	return ok && status == "healthy", nil
}

func (c *Client) doRequest(method, url string) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "casman-cli")

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to extract error message
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if msg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("server error: %s", msg)
			}
		}
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return body, nil
}
