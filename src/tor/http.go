package tor

import (
	"context"
	"net/http"
	"time"
)

// HTTPTransport returns an *http.Transport whose connections are dialed via
// the bundled Tor SOCKS5 proxy. The transport disables HTTP/2 because the
// SOCKS dialer doesn't expose ALPN, and uses conservative timeouts since
// Tor latency varies. Returns nil + error when Tor isn't running.
func (s *Service) HTTPTransport() (*http.Transport, error) {
	d, err := s.dialer(context.Background())
	if err != nil {
		return nil, err
	}
	return &http.Transport{
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          16,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		DialContext:           d.DialContext,
	}, nil
}

// HTTPClient returns an *http.Client whose Transport routes through Tor.
// `timeout` is the per-request total timeout; pass 0 for no timeout.
func (s *Service) HTTPClient(timeout time.Duration) (*http.Client, error) {
	tr, err := s.HTTPTransport()
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: tr, Timeout: timeout}, nil
}
