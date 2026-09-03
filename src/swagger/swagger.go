// Package swagger provides OpenAPI/Swagger specification and UI for casman.
// See AI.md for details.
package swagger

import (
	"encoding/json"
	"net/http"
)

// Spec holds the OpenAPI specification.
type Spec struct {
	OpenAPI string                 `json:"openapi"`
	Info    Info                   `json:"info"`
	Servers []Server               `json:"servers"`
	Paths   map[string]PathItem    `json:"paths"`
	Tags    []Tag                  `json:"tags,omitempty"`
}

// Info holds API information.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// Server holds server information.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Tag holds tag information.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PathItem holds path operations.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation holds operation details.
type Operation struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

// Parameter holds parameter details.
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Schema      Schema `json:"schema"`
}

// Schema holds schema details.
type Schema struct {
	Type string `json:"type"`
}

// Response holds response details.
type Response struct {
	Description string `json:"description"`
}

// Handler provides HTTP handlers for Swagger UI and spec.
type Handler struct {
	spec    *Spec
	version string
	fqdn    string
}

// New creates a new Swagger handler.
func New(version, fqdn string) *Handler {
	h := &Handler{
		version: version,
		fqdn:    fqdn,
	}
	h.spec = h.generateSpec()
	return h
}

// generateSpec creates the OpenAPI specification.
func (h *Handler) generateSpec() *Spec {
	return &Spec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "casman API",
			Description: "Universal Man Page API",
			Version:     h.version,
		},
		Servers: []Server{
			{URL: h.fqdn, Description: "Production server"},
		},
		Tags: []Tag{
			{Name: "Health", Description: "Health check endpoints"},
			{Name: "Man Pages", Description: "Man page retrieval"},
			{Name: "Search", Description: "Search functionality"},
			{Name: "Browse", Description: "Browse man pages"},
			{Name: "Compare", Description: "Compare pages across platforms"},
		},
		Paths: map[string]PathItem{
			"/api/v1/healthz": {
				Get: &Operation{
					Summary: "Health check",
					Tags:    []string{"Health"},
					Responses: map[string]Response{
						"200": {Description: "Healthy"},
					},
				},
			},
			"/api/v1/stats": {
				Get: &Operation{
					Summary: "Get database statistics",
					Tags:    []string{"Health"},
					Responses: map[string]Response{
						"200": {Description: "Statistics"},
					},
				},
			},
			"/api/v1/sections": {
				Get: &Operation{
					Summary: "List all sections",
					Tags:    []string{"Browse"},
					Responses: map[string]Response{
						"200": {Description: "List of sections"},
					},
				},
			},
			"/api/v1/platforms": {
				Get: &Operation{
					Summary: "List all platforms",
					Tags:    []string{"Browse"},
					Responses: map[string]Response{
						"200": {Description: "List of platforms"},
					},
				},
			},
			"/api/v1/search": {
				Get: &Operation{
					Summary: "Search man pages",
					Tags:    []string{"Search"},
					Parameters: []Parameter{
						{Name: "q", In: "query", Required: true, Schema: Schema{Type: "string"}, Description: "Search query"},
						{Name: "section", In: "query", Schema: Schema{Type: "string"}, Description: "Filter by section"},
						{Name: "platform", In: "query", Schema: Schema{Type: "string"}, Description: "Filter by platform"},
						{Name: "page", In: "query", Schema: Schema{Type: "integer"}, Description: "Page number"},
					},
					Responses: map[string]Response{
						"200": {Description: "Search results"},
					},
				},
			},
			"/api/v1/man/{name}": {
				Get: &Operation{
					Summary: "Get man page by name",
					Tags:    []string{"Man Pages"},
					Parameters: []Parameter{
						{Name: "name", In: "path", Required: true, Schema: Schema{Type: "string"}},
					},
					Responses: map[string]Response{
						"200": {Description: "Man page"},
						"404": {Description: "Not found"},
					},
				},
			},
			"/api/v1/man/{section}/{name}": {
				Get: &Operation{
					Summary: "Get man page by section and name",
					Tags:    []string{"Man Pages"},
					Parameters: []Parameter{
						{Name: "section", In: "path", Required: true, Schema: Schema{Type: "string"}},
						{Name: "name", In: "path", Required: true, Schema: Schema{Type: "string"}},
					},
					Responses: map[string]Response{
						"200": {Description: "Man page"},
						"404": {Description: "Not found"},
					},
				},
			},
			"/api/v1/man/{os}/{section}/{name}": {
				Get: &Operation{
					Summary: "Get man page by OS, section, and name",
					Tags:    []string{"Man Pages"},
					Parameters: []Parameter{
						{Name: "os", In: "path", Required: true, Schema: Schema{Type: "string"}},
						{Name: "section", In: "path", Required: true, Schema: Schema{Type: "string"}},
						{Name: "name", In: "path", Required: true, Schema: Schema{Type: "string"}},
					},
					Responses: map[string]Response{
						"200": {Description: "Man page"},
						"404": {Description: "Not found"},
					},
				},
			},
			"/api/v1/compare/{name}": {
				Get: &Operation{
					Summary: "Compare man page across platforms",
					Tags:    []string{"Compare"},
					Parameters: []Parameter{
						{Name: "name", In: "path", Required: true, Schema: Schema{Type: "string"}},
					},
					Responses: map[string]Response{
						"200": {Description: "Comparison result"},
						"404": {Description: "Not found"},
					},
				},
			},
			"/api/v1/whatis/{name}": {
				Get: &Operation{
					Summary: "Get whatis description",
					Tags:    []string{"Man Pages"},
					Parameters: []Parameter{
						{Name: "name", In: "path", Required: true, Schema: Schema{Type: "string"}},
					},
					Responses: map[string]Response{
						"200": {Description: "Whatis result"},
						"404": {Description: "Not found"},
					},
				},
			},
			"/api/v1/apropos": {
				Get: &Operation{
					Summary: "Search man page descriptions",
					Tags:    []string{"Search"},
					Parameters: []Parameter{
						{Name: "q", In: "query", Required: true, Schema: Schema{Type: "string"}},
					},
					Responses: map[string]Response{
						"200": {Description: "Apropos results"},
					},
				},
			},
		},
	}
}

// ServeSpec returns the OpenAPI specification as JSON.
func (h *Handler) ServeSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(h.spec)
}

// ServeUI returns the Swagger UI HTML page.
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerUIHTML))
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>casman API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        .swagger-ui .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: "/api/v1/openapi",
            dom_id: '#swagger-ui',
            presets: [SwaggerUIBundle.presets.apis],
            layout: "BaseLayout"
        });
    </script>
</body>
</html>
`
