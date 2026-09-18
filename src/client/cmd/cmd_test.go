package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casapps/casman/src/client/api"
)

func TestNew(t *testing.T) {
	cfg := &Config{ServerURL: "http://example.com", Token: "tok"}
	r := New(cfg)
	if r.client == nil {
		t.Fatal("client is nil")
	}
	if r.config != cfg {
		t.Error("config not stored")
	}
}

func newTestRunner(t *testing.T, handler http.HandlerFunc) *Runner {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Runner{
		client: api.New(srv.URL, ""),
		config: &Config{},
	}
}

func TestMan_ByNameOnly(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/man/ls" {
			t.Errorf("path = %q", req.URL.Path)
		}
		json.NewEncoder(w).Encode(api.ManPage{Name: "ls", Section: "1", Title: "list files", ContentText: "usage: ls"})
	})

	if err := r.Man("ls", ""); err != nil {
		t.Fatalf("Man: %v", err)
	}
}

func TestMan_WithSection(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/man/1/ls" {
			t.Errorf("path = %q", req.URL.Path)
		}
		json.NewEncoder(w).Encode(api.ManPage{Name: "ls", Section: "1", ContentText: "usage: ls"})
	})

	if err := r.Man("ls", "1"); err != nil {
		t.Fatalf("Man: %v", err)
	}
}

func TestMan_FallsBackToHTML(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(api.ManPage{Name: "ls", Section: "1", ContentHTML: "<p>usage</p>"})
	})

	if err := r.Man("ls", ""); err != nil {
		t.Fatalf("Man: %v", err)
	}
}

func TestMan_FallsBackToMarkdown(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(api.ManPage{Name: "ls", Section: "1", ContentMarkdown: "# usage"})
	})

	if err := r.Man("ls", ""); err != nil {
		t.Fatalf("Man: %v", err)
	}
}

func TestMan_NoContent(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(api.ManPage{Name: "ls", Section: "1"})
	})

	if err := r.Man("ls", ""); err == nil {
		t.Fatal("expected error for missing content, got nil")
	}
}

func TestMan_FetchError(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})

	if err := r.Man("nonexistent", ""); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSearch_WithResults(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(api.SearchResponse{
			Query: "ls",
			Total: 2,
			Results: []api.SearchResult{
				{Name: "ls", Section: "1", Title: "list", Snippet: "<mark>ls</mark> lists files"},
			},
		})
	})

	if err := r.Search("ls", "", ""); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestSearch_NoResults(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(api.SearchResponse{Query: "zzz", Total: 0})
	})

	if err := r.Search("zzz", "", ""); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestSearch_Error(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := r.Search("ls", "", ""); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWhatis_WithResults(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode([]api.SearchResult{{Name: "ls", Section: "1", Title: "list files"}})
	})

	if err := r.Whatis("ls"); err != nil {
		t.Fatalf("Whatis: %v", err)
	}
}

func TestWhatis_NoResults(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode([]api.SearchResult{})
	})

	if err := r.Whatis("nonexistent"); err != nil {
		t.Fatalf("Whatis: %v", err)
	}
}

func TestWhatis_Error(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := r.Whatis("ls"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestApropos_WithResults(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   "list",
			"results": []api.SearchResult{{Name: "ls", Section: "1", Title: "list files"}},
		})
	})

	if err := r.Apropos("list"); err != nil {
		t.Fatalf("Apropos: %v", err)
	}
}

func TestApropos_NoResults(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"query": "zzz", "results": []api.SearchResult{}})
	})

	if err := r.Apropos("zzz"); err != nil {
		t.Fatalf("Apropos: %v", err)
	}
}

func TestStats(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(api.Stats{
			TotalPages:     10,
			TotalSections:  3,
			TotalPlatforms: 2,
			TotalLanguages: 1,
			BySection:      map[string]int{"1": 5},
			ByPlatform:     map[string]int{"linux": 8},
		})
	})

	if err := r.Stats(); err != nil {
		t.Fatalf("Stats: %v", err)
	}
}

func TestStats_Error(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := r.Stats(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSections(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode([]api.Section{
			{ID: "1", Name: "General Commands", Count: 5},
			{ID: "5", Name: "File Formats"},
		})
	})

	if err := r.Sections(); err != nil {
		t.Fatalf("Sections: %v", err)
	}
}

func TestSections_Error(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := r.Sections(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPlatforms(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode([]api.Platform{
			{ID: "linux", Name: "Linux", Count: 5},
			{ID: "macos", Name: "macOS"},
		})
	})

	if err := r.Platforms(); err != nil {
		t.Fatalf("Platforms: %v", err)
	}
}

func TestPlatforms_Error(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := r.Platforms(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	if err := r.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "degraded"})
	})

	if err := r.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}

func TestHealthCheck_Error(t *testing.T) {
	r := newTestRunner(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := r.HealthCheck(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDisplayWithPager_NoPagerFallsBackToPrint(t *testing.T) {
	r := &Runner{config: &Config{Pager: true}}
	if err := r.displayWithPager("hello world"); err != nil {
		t.Fatalf("displayWithPager: %v", err)
	}
}

func TestStripHTML(t *testing.T) {
	cases := map[string]string{
		"<p>Hello <b>world</b></p>":                      "Hello world",
		"<script>alert(1)</script>Visible text":           "Visible text",
		"<style>.a{color:red}</style>Visible":              "Visible",
		"A&nbsp;B &amp; C &lt;tag&gt; &quot;q&quot; &#39;s&#39;": "A B & C <tag> \"q\" 's'",
		"  multiple   spaces  ":                           "multiple spaces",
		"":                                                "",
	}
	for in, want := range cases {
		if got := stripHTML(in); got != want {
			t.Errorf("stripHTML(%q) = %q, want %q", in, got, want)
		}
	}
}
