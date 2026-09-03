// Package graphql provides GraphQL API for casman.
// See AI.md for details.
package graphql

import (
	"encoding/json"
	"net/http"
)

// Handler provides HTTP handlers for GraphQL API and UI.
type Handler struct {
	version string
}

// New creates a new GraphQL handler.
func New(version string) *Handler {
	return &Handler{
		version: version,
	}
}

// Request represents a GraphQL request.
type Request struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// Response represents a GraphQL response.
type Response struct {
	Data   interface{}   `json:"data,omitempty"`
	Errors []GraphError  `json:"errors,omitempty"`
}

// GraphError represents a GraphQL error.
type GraphError struct {
	Message string `json:"message"`
}

// ServeGraphQL handles GraphQL requests.
func (h *Handler) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	var req Request

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "Invalid request body")
			return
		}
	} else if r.Method == "GET" {
		req.Query = r.URL.Query().Get("query")
		req.OperationName = r.URL.Query().Get("operationName")
	}

	if req.Query == "" {
		writeError(w, "Query is required")
		return
	}

	// Execute query
	result := h.executeQuery(req)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(result)
}

// executeQuery executes a GraphQL query.
func (h *Handler) executeQuery(req Request) Response {
	// Basic introspection support
	if req.Query == IntrospectionQuery {
		return Response{
			Data: map[string]interface{}{
				"__schema": h.getSchema(),
			},
		}
	}

	// For now, return schema info
	return Response{
		Data: map[string]interface{}{
			"version": h.version,
		},
	}
}

// getSchema returns the GraphQL schema for introspection.
func (h *Handler) getSchema() map[string]interface{} {
	return map[string]interface{}{
		"queryType": map[string]interface{}{
			"name": "Query",
		},
		"types": []map[string]interface{}{
			{
				"name": "Query",
				"kind": "OBJECT",
				"fields": []map[string]interface{}{
					{"name": "manPage", "description": "Get a man page"},
					{"name": "search", "description": "Search man pages"},
					{"name": "sections", "description": "List sections"},
					{"name": "platforms", "description": "List platforms"},
					{"name": "compare", "description": "Compare page across platforms"},
					{"name": "stats", "description": "Get statistics"},
				},
			},
			{
				"name": "ManPage",
				"kind": "OBJECT",
				"fields": []map[string]interface{}{
					{"name": "name", "description": "Page name"},
					{"name": "section", "description": "Section number"},
					{"name": "title", "description": "Page title"},
					{"name": "platform", "description": "Platform/OS"},
					{"name": "synopsis", "description": "Command synopsis"},
					{"name": "contentHTML", "description": "HTML content"},
					{"name": "seeAlso", "description": "Related pages"},
				},
			},
			{
				"name": "SearchResult",
				"kind": "OBJECT",
				"fields": []map[string]interface{}{
					{"name": "name", "description": "Page name"},
					{"name": "section", "description": "Section number"},
					{"name": "title", "description": "Page title"},
					{"name": "platform", "description": "Platform/OS"},
					{"name": "snippet", "description": "Matching snippet"},
					{"name": "score", "description": "Relevance score"},
				},
			},
			{
				"name": "Section",
				"kind": "OBJECT",
				"fields": []map[string]interface{}{
					{"name": "id", "description": "Section ID"},
					{"name": "name", "description": "Section name"},
					{"name": "description", "description": "Description"},
					{"name": "count", "description": "Page count"},
				},
			},
			{
				"name": "Platform",
				"kind": "OBJECT",
				"fields": []map[string]interface{}{
					{"name": "id", "description": "Platform ID"},
					{"name": "name", "description": "Platform name"},
					{"name": "description", "description": "Description"},
					{"name": "count", "description": "Page count"},
				},
			},
		},
	}
}

// ServeUI returns the GraphiQL UI HTML page.
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(graphiqlHTML))
}

func writeError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(Response{
		Errors: []GraphError{{Message: message}},
	})
}

// IntrospectionQuery is the standard GraphQL introspection query.
const IntrospectionQuery = `
  query IntrospectionQuery {
    __schema {
      queryType { name }
      types { name kind description }
    }
  }
`

const graphiqlHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>casman GraphQL API</title>
    <link href="https://unpkg.com/graphiql@3/graphiql.min.css" rel="stylesheet" />
    <style>
        body { margin: 0; overflow: hidden; height: 100vh; }
        #graphiql { height: 100vh; }
    </style>
</head>
<body>
    <div id="graphiql"></div>
    <script crossorigin src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
    <script src="https://unpkg.com/graphiql@3/graphiql.min.js"></script>
    <script>
        const fetcher = GraphiQL.createFetcher({ url: '/graphql' });
        ReactDOM.createRoot(document.getElementById('graphiql')).render(
            React.createElement(GraphiQL, { fetcher: fetcher })
        );
    </script>
</body>
</html>
`
