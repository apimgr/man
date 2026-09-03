// Inline HTTP-01 challenge provider. Lego's default http01.NewProviderServer
// binds its own listener on port 80, which conflicts with our long-running
// redirect listener. This provider instead exposes an http.Handler that the
// listener can mount under /.well-known/acme-challenge/, while implementing
// challenge.Provider so lego stores tokens here. See AI.md PART 15.

package ssl

import (
	"net/http"
	"strings"
	"sync"
)

const challengePathPrefix = "/.well-known/acme-challenge/"

// httpChallenge buffers the active HTTP-01 token→keyAuth pairs.
type httpChallenge struct {
	mu     sync.RWMutex
	tokens map[string]string
}

func newHTTPChallenge() *httpChallenge {
	return &httpChallenge{tokens: map[string]string{}}
}

// Present satisfies challenge.Provider. Lego calls this once per challenge.
func (h *httpChallenge) Present(domain, token, keyAuth string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tokens[token] = keyAuth
	return nil
}

// CleanUp satisfies challenge.Provider.
func (h *httpChallenge) CleanUp(domain, token, keyAuth string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.tokens, token)
	return nil
}

// ServeHTTP responds to /.well-known/acme-challenge/{token} requests with the
// stored key authorization. Unknown tokens return 404.
func (h *httpChallenge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, challengePathPrefix) {
		http.NotFound(w, r)
		return
	}
	token := strings.TrimPrefix(r.URL.Path, challengePathPrefix)
	if token == "" || strings.Contains(token, "/") {
		http.NotFound(w, r)
		return
	}
	h.mu.RLock()
	keyAuth, ok := h.tokens[token]
	h.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(keyAuth))
}

// HTTPChallengeHandler exposes the challenge HTTP handler so the listener can
// mount it ahead of the redirect handler.
func (p *Provisioner) HTTPChallengeHandler() http.Handler {
	if p == nil || p.httpChallenge == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.NotFound(w, nil)
		})
	}
	return p.httpChallenge
}
