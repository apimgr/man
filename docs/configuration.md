# Configuration

## Config File

The file is named `server.yml` in the selected configuration directory. The
default directory is `/etc/casapps/casman` for root, `/config` in a container,
and the platform's user configuration directory for an unprivileged process.
The `--config` and `--data` flags override the corresponding directories.

```yaml
server:
  address: "[::]"
  port: ""
  https_port: "443"
  http_redirect_port: "80"
  fqdn: localhost
  mode: production
  branding:
    title: casman
    tagline: Universal Man Pages
    description: Man pages from BSD, macOS, Linux, and more
  ssl:
    enabled: false
    min_version: TLS1.2
    cert: ""
    key: ""
    data_dir: ""
    letsencrypt:
      enabled: false
      email: ""
      challenge: http-01
      staging: false

database:
  driver: sqlite
  path: "{data_dir}/db/server.db"
```

## SSL/TLS

Per AI.md PART 15, casman ships built-in Let's Encrypt support and an
encrypted DNS provider credential vault. The wiring is:

- Set `server.ssl.enabled: true` to switch the listener stack from HTTP-only to
  HTTP redirect + HTTPS. The HTTPS listener uses a tightened TLS profile
  (TLS 1.2+, ECDHE+AES/ChaCha20, X25519/P256).
- For an operator-supplied cert, set `server.ssl.cert` and `server.ssl.key`. The cert is
  loaded into the in-memory cache at startup and served via SNI.
- For automated issuance, set `server.ssl.letsencrypt.enabled: true`,
  `server.ssl.letsencrypt.email` and `server.ssl.letsencrypt.challenge` to one of `http-01`,
  `tls-alpn-01`, or `dns-01`. Use `staging: true` while testing to avoid
  Let's Encrypt rate limits.
- A DNS credential vault is present, but this release does not expose a
  supported web or CLI workflow for adding DNS provider credentials. Use
  HTTP-01 or an operator-supplied certificate until that management path is
  implemented. The code includes provider adapters for Cloudflare, Route 53,
  DigitalOcean, GoDaddy, Namecheap, RFC 2136, and a manual fallback.
- A daily scheduler task (03:00) parses each cached cert's `NotAfter` and
  re-provisions any cert with less than 30 days of life remaining. Static
  (non-ACME) certs are inspected too so operators get an early-warning
  log entry before expiry.
- Override the master key by setting `CASMAN_MASTER_SECRET` to a
  hex-encoded 32-byte value; this skips the disk file entirely and is
  recommended in container deployments.
- HTTP-01 shares port 80 with the redirect listener (no second bind).
  TLS-ALPN-01 uses lego's own listener and therefore conflicts with our
  HTTPS server — prefer HTTP-01 or DNS-01 in normal operation.

## Environment Variables

Environment variables may override settings supported by the server entry
point. Common examples include:

```bash
MODE=development
PORT=8080
DEBUG=true
```

## CLI Flags

```bash
casman --port 8080 --mode development --debug
```

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Listen port (HTTP) | Environment/config dependent |
| `--address` | Listen address | `[::]` |
| `--https-port` | Listen port (HTTPS, when SSL is enabled) | 443 |
| `--http-redirect-port` | Listen port (HTTP redirect + ACME) | 80 |
| `--mode` | production/development | production |
| `--config` | Config directory | OS-specific |
| `--data` | Data directory | OS-specific |
| `--debug` | Enable debug mode | false |

## Optional features

The server configuration also supports metrics, GeoIP, backup, and Tor
settings. These sections are optional; see the comments and defaults in
`src/config/config.go`. There is no admin web UI. TLS and DNS provider
credentials must be configured through the server's supported configuration
and CLI workflows.
