package ssl

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPChallengeServesPresentedToken(t *testing.T) {
	h := newHTTPChallenge()
	if err := h.Present("example.com", "tok", "tok.auth"); err != nil {
		t.Fatalf("Present() error = %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, challengePathPrefix+"tok", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "tok.auth" {
		t.Fatalf("got %d %q, want 200 tok.auth", rec.Code, rec.Body.String())
	}
	if err := h.CleanUp("example.com", "tok", "tok.auth"); err != nil {
		t.Fatalf("CleanUp() error = %v", err)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, challengePathPrefix+"tok", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("after CleanUp got %d, want 404", rec.Code)
	}
}

func TestHTTPChallengeRejectsBadPaths(t *testing.T) {
	h := newHTTPChallenge()
	for _, path := range []string{"/other", challengePathPrefix, challengePathPrefix + "a/b"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%q got %d, want 404", path, rec.Code)
		}
	}
}

func TestHTTPChallengeHandlerReturnsProviderHandler(t *testing.T) {
	h := newHTTPChallenge()
	p := &Provisioner{httpChallenge: h}
	if p.HTTPChallengeHandler() != http.Handler(h) {
		t.Fatal("HTTPChallengeHandler() did not return the challenge handler")
	}
	rec := httptest.NewRecorder()
	(*Provisioner)(nil).HTTPChallengeHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("nil provisioner handler got %d, want 404", rec.Code)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	data, err := encodeJSON(map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("encodeJSON() error = %v", err)
	}
	var out map[string]string
	if err := decodeJSON(data, &out); err != nil || out["a"] != "b" {
		t.Fatalf("decodeJSON() = %v, %v", out, err)
	}
}
