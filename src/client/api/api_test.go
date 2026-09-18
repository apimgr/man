package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_DefaultServerURL(t *testing.T) {
	c := New("", "tok")
	if c.ServerURL != "http://localhost:64580" {
		t.Errorf("ServerURL = %q, want default", c.ServerURL)
	}
	if c.Token != "tok" {
		t.Errorf("Token = %q, want tok", c.Token)
	}
}

func TestNew_CustomServerURL(t *testing.T) {
	c := New("http://example.com", "")
	if c.ServerURL != "http://example.com" {
		t.Errorf("ServerURL = %q, want http://example.com", c.ServerURL)
	}
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-token")
}

func TestGetManPage(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing/incorrect Authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/man/ls" {
			t.Errorf("path = %q, want /api/v1/man/ls", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ManPage{Name: "ls", Section: "1"})
	})

	page, err := c.GetManPage("ls")
	if err != nil {
		t.Fatalf("GetManPage: %v", err)
	}
	if page.Name != "ls" {
		t.Errorf("Name = %q, want ls", page.Name)
	}
}

func TestGetManPageSection(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/man/1/ls" {
			t.Errorf("path = %q, want /api/v1/man/1/ls", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ManPage{Name: "ls", Section: "1"})
	})

	page, err := c.GetManPageSection("1", "ls")
	if err != nil {
		t.Fatalf("GetManPageSection: %v", err)
	}
	if page.Section != "1" {
		t.Errorf("Section = %q, want 1", page.Section)
	}
}

func TestGetManPagePlatform(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/man/linux/1/ls" {
			t.Errorf("path = %q, want /api/v1/man/linux/1/ls", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ManPage{Name: "ls", Platform: "linux"})
	})

	page, err := c.GetManPagePlatform("linux", "1", "ls")
	if err != nil {
		t.Fatalf("GetManPagePlatform: %v", err)
	}
	if page.Platform != "linux" {
		t.Errorf("Platform = %q, want linux", page.Platform)
	}
}

func TestGetManPage_NotFound(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})

	_, err := c.GetManPage("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSearch(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "ls" {
			t.Errorf("q = %q, want ls", r.URL.Query().Get("q"))
		}
		json.NewEncoder(w).Encode(SearchResponse{Query: "ls", Total: 1})
	})

	resp, err := c.Search("ls", "", "", 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
}

func TestSearch_WithFilters(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("section") != "1" || q.Get("platform") != "linux" || q.Get("page") != "2" {
			t.Errorf("unexpected query: %v", q)
		}
		json.NewEncoder(w).Encode(SearchResponse{})
	})

	if _, err := c.Search("ls", "1", "linux", 2); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestWhatis(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]SearchResult{{Name: "ls", Section: "1"}})
	})

	results, err := c.Whatis("ls")
	if err != nil {
		t.Fatalf("Whatis: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v, want 1 entry", results)
	}
}

func TestApropos(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   "list",
			"results": []SearchResult{{Name: "ls"}},
		})
	})

	results, err := c.Apropos("list")
	if err != nil {
		t.Fatalf("Apropos: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v, want 1 entry", results)
	}
}

func TestGetStats(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Stats{})
	})

	if _, err := c.GetStats(); err != nil {
		t.Fatalf("GetStats: %v", err)
	}
}

func TestGetSections(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Section{{ID: "1"}})
	})

	sections, err := c.GetSections()
	if err != nil {
		t.Fatalf("GetSections: %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("sections = %+v, want 1 entry", sections)
	}
}

func TestGetPlatforms(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Platform{{ID: "linux"}})
	})

	platforms, err := c.GetPlatforms()
	if err != nil {
		t.Fatalf("GetPlatforms: %v", err)
	}
	if len(platforms) != 1 {
		t.Fatalf("platforms = %+v, want 1 entry", platforms)
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	ok, err := c.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if !ok {
		t.Error("HealthCheck = false, want true")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "degraded"})
	})

	ok, err := c.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if ok {
		t.Error("HealthCheck = true, want false")
	}
}

func TestDoRequest_ServerErrorWithMessage(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
	})

	_, err := c.GetStats()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDoRequest_ServerErrorNoBody(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetStats()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDoRequest_NoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization header set without token: %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(Stats{})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.GetStats(); err != nil {
		t.Fatalf("GetStats: %v", err)
	}
}
