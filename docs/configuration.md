# Configuration

## Config File

Default location: `/etc/casapps/casman/server.yml`

```yaml
server:
  address: 0.0.0.0
  port: 80
  mode: production

database:
  driver: sqlite
  path: /data/db/server.db

ssl:
  enabled: false
  cert: ""
  key: ""
  data_dir: ""
  letsencrypt:
    enabled: false
    email: admin@example.com
    challenge: http-01
    staging: false
```

## SSL/TLS

Per AI.md PART 15, casman ships built-in Let's Encrypt support and an
encrypted DNS provider credential vault. The wiring is:

- Set `ssl.enabled: true` to switch the listener stack from HTTP-only to
  HTTP redirect + HTTPS. The HTTPS listener uses a tightened TLS profile
  (TLS 1.2+, ECDHE+AES/ChaCha20, X25519/P256).
- For an operator-supplied cert, set `ssl.cert` and `ssl.key`. The cert is
  loaded into the in-memory cache at startup and served via SNI.
- For automated issuance, set `ssl.letsencrypt.enabled: true`,
  `letsencrypt.email`, and `letsencrypt.challenge` to one of `http-01`,
  `tls-alpn-01`, or `dns-01`. Use `staging: true` while testing to avoid
  Let's Encrypt rate limits.
- For DNS-01, configure provider credentials at
  `Admin → Server → SSL/TLS`. Credentials are encrypted with the server
  master key (`{config_dir}/security/master.key`, 0600) before they hit
  the database. The supported providers track lego's: cloudflare, route53,
  digitalocean, godaddy, namecheap, rfc2136, plus a `manual` fallback.
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

All settings can be overridden via environment:

```bash
CASMAN_SERVER_PORT=8080
CASMAN_SERVER_MODE=development
CASMAN_DATABASE_DRIVER=postgres
```

## CLI Flags

```bash
casman --port 8080 --mode development --debug
```

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Listen port (HTTP) | 80 (container), random 64xxx (host) |
| `--address` | Listen address | 0.0.0.0 |
| `--https-port` | Listen port (HTTPS, when ssl.enabled) | 443 |
| `--http-redirect-port` | Listen port (HTTP→HTTPS redirect + ACME) | 80 |
| `--mode` | production/development | production |
| `--config` | Config directory | OS-specific |
| `--data` | Data directory | OS-specific |
| `--debug` | Enable debug mode | false |

## Admin Panel

All settings are configurable via the WebUI at `/admin`.
