package graphql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeGraphQL_POST(t *testing.T) {
	h := New("1.0.0")
	body := strings.NewReader(`{"query":"{ version }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	w := httptest.NewRecorder()

	h.ServeGraphQL(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Errors != nil {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}
}

func TestServeGraphQL_GET(t *testing.T) {
	h := New("1.0.0")
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={version}", nil)
	w := httptest.NewRecorder()

	h.ServeGraphQL(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestServeGraphQL_MissingQuery(t *testing.T) {
	h := New("1.0.0")
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	w := httptest.NewRecorder()

	h.ServeGraphQL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected error for missing query")
	}
}

func TestServeGraphQL_InvalidBody(t *testing.T) {
	h := New("1.0.0")
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	h.ServeGraphQL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestServeGraphQL_Introspection(t *testing.T) {
	h := New("1.0.0")
	reqBody, _ := json.Marshal(Request{Query: IntrospectionQuery})
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(reqBody)))
	w := httptest.NewRecorder()

	h.ServeGraphQL(w, req)

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}
	if _, ok := data["__schema"]; !ok {
		t.Error("expected __schema in introspection response")
	}
}

func TestServeUI(t *testing.T) {
	h := New("1.0.0")
	req := httptest.NewRequest(http.MethodGet, "/graphiql", nil)
	w := httptest.NewRecorder()

	h.ServeUI(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	if !strings.Contains(w.Body.String(), "graphiql") {
		t.Error("expected graphiql markup in body")
	}
}

func TestGetSchema(t *testing.T) {
	h := New("1.0.0")
	schema := h.getSchema()
	if schema["queryType"] == nil {
		t.Error("expected queryType in schema")
	}
	types, ok := schema["types"].([]map[string]interface{})
	if !ok || len(types) == 0 {
		t.Error("expected non-empty types list")
	}
}
