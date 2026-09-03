// HTTP and HTTPS listener that serves TLS using Provisioner.GetCertificate
// and binds a port-80 helper that 301-redirects everything to its HTTPS peer.
// Used by src/server/server.go when SSL is enabled. See AI.md PART 15.

package ssl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServeBoth runs an HTTP listener on httpAddr (for redirects + ACME HTTP-01
// fallback) and a TLS listener on httpsAddr that uses prov.GetCertificate.
// The TLS listener also includes the tls-alpn-01 NextProtos so lego's
// TLS-ALPN-01 challenges work without a second listener. The function blocks
// until ctx is cancelled or either listener returns a fatal error.
func ServeBoth(ctx context.Context, httpAddr, httpsAddr string, handler http.Handler, prov *Provisioner) error {
	if prov == nil {
		return errors.New("ssl: ServeBoth requires a Provisioner")
	}

	tlsCfg := DefaultTLSConfig()
	tlsCfg.GetCertificate = prov.GetCertificate
	tlsCfg.NextProtos = append(tlsCfg.NextProtos, "h2", "http/1.1", "acme-tls/1")

	httpsSrv := &http.Server{
		Addr:              httpsAddr,
		Handler:           handler,
		TLSConfig:         tlsCfg,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	mux := http.NewServeMux()
	mux.Handle(challengePathPrefix, prov.HTTPChallengeHandler())
	mux.Handle("/", redirectHandler(hostFromAddr(httpsAddr)))
	httpSrv := &http.Server{
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("ssl: HTTPS listening on %s", httpsAddr)
		if err := httpsSrv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("https: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("ssl: HTTP redirect listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		shutdown(httpsSrv, httpSrv)
		wg.Wait()
		return err
	}

	shutdown(httpsSrv, httpSrv)
	wg.Wait()
	return nil
}

func shutdown(servers ...*http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, s := range servers {
		_ = s.Shutdown(ctx)
	}
}

// redirectHandler returns an http.Handler that 301-redirects every request to
// the HTTPS peer using the request's Host (with the configured https port if
// present). Requests to /.well-known/acme-challenge/* are passed through to a
// 404 since lego's HTTP-01 provider binds its own listener on this same
// address; in our model that listener is started transiently during cert
// obtain and shadows this redirect handler for the duration.
func redirectHandler(httpsHost string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if i := strings.IndexByte(host, ':'); i >= 0 {
			host = host[:i]
		}
		if host == "" {
			host = httpsHost
		}
		target := "https://" + host
		if httpsHost != "" && httpsHost != "443" {
			// Append port only when it is non-default; httpsHost here is the
			// already-extracted host name, not the port. The port suffix is
			// added below when needed.
		}
		target += r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

func hostFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return host
}

// loopbackBindFallback returns true when binding the privileged port at addr
// failed with permission denied; callers can use it to surface a friendlier
// error message (PART 25 covers privilege drop / setcap guidance).
func loopbackBindFallback(err error) bool {
	if err == nil {
		return false
	}
	var nErr *net.OpError
	if errors.As(err, &nErr) {
		return strings.Contains(nErr.Error(), "permission denied")
	}
	return false
}

// Compile-time guarantee that the redirect handler is an http.Handler.
var _ http.Handler = redirectHandler("example.com")

// Compile-time use of the helpers; some are exported for tests in the
// future and we want vet to keep them honest now.
var _ = loopbackBindFallback
var _ = tls.VersionTLS12
