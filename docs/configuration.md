# Configuration

## Config File

The file is named `server.yml` in the selected configuration directory. The
default directory is `/etc/casapps/casman` for root, `/config` in a container,
and the platform's user configuration directory for an unprivileged process.
The `--config` and `--data` flags override the corresponding directories.
Configuration merges built-in defaults, `server.yml`, byte-wise sorted
`yaml.d/*.yml` and `yaml.d/*.yaml` drop-ins, then `secrets.yml`. Maps merge
recursively; a later scalar or list replaces the earlier value. Null,
duplicate, or wrongly typed keys reset to defaults with a file and line
warning. An unparseable file is ignored. Config files use mode `0600` and
`yaml.d/` uses mode `0700`.

```yaml
server:
  address: "[::]"
  port: "64580"
  fqdn: localhost
  mode: production
  api_version: v1
  database:
    driver: sqlite
    url: "{data_dir}/db/server.db"
  cache:
    type: memory
    url: ""
    host: localhost
    port: 6379
    username: ""
    password: ""
    db: 0
    tls: false
    tls_skip_verify: true
    tls_ca: ""
    tls_cert: ""
    tls_key: ""
    pool_size: 10
    min_idle: 2
    timeout: 5s
    prefix: casman:
    ttl: 1h
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

```

`server.yml` holds this node's listen address, identity, database, cache, and
SSL settings. Branding and SEO belong in `yaml.d/branding.yml`; web settings
and CORS belong in `yaml.d/web.yml`; rate limits, tokens, and security settings
belong in `yaml.d/security.yml`; report and help pages belong in
`yaml.d/pages.yml`; and `web.*` LLM discovery options belong in
`yaml.d/web.yml`. Credentials belong in `secrets.yml`. Every file uses the
same dotted YAML paths as the merged configuration.

The server migrates a legacy `server.yaml` to `server.yml` and splits known
area settings from a large `server.yml` on startup. It keeps a private
`server.yml.bak`; settings already present in an area file take precedence.

`server.api_version` defaults to `v1`; set it to a positive version such as
`v2` to move the canonical API routes, OpenAPI document, MCP endpoint, and
generated links to `/api/v2`. Keep the CLI's `server.api_version` setting in
`cli.yml` in sync with the server.

`server.cache.type` defaults to `memory`; choose `none`, `valkey`, or `redis`
to change it. For an external cache, configure either `url` or the host,
port, credentials, and database fields. `rediss://` and `valkeys://` URLs
enable TLS; `tls_ca`, `tls_cert`, and `tls_key` name files in
`{config_dir}/ssl/`. A cache URL takes precedence over individual connection
fields. Rate-limit counters remain in SQLite unless an external cache is
selected.

`server.rate_limit` configures per-IP or per-Bearer-token read and write
windows, a health endpoint window, a stricter owner-token action window, and
an absolute per-IP burst ceiling. Repeated invalid Bearer credentials are
counted per IP and lock that IP out for the `auth_failures.window` duration
after `auth_failures.requests` failures. Windows are seconds.

`server.limits` belongs in `yaml.d/security.yml`:

```yaml
server:
  limits:
    max_body_size: 10MB
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s
```

Body-size values use byte units (`B`, `KB`, `MB`, `GB`, or `TB`) with 1024-based
scaling. Missing, malformed, zero, and negative values use the defaults above
and emit a warning. A valid body-size change applies to new requests on reload.
Timeout changes replace the active HTTP server over the existing listener and
gracefully drain existing requests without a restart.

`web.llms.include_endpoints` controls whether `llms.txt` lists API operations
(default `true`). `include_schemas` adds the available parameter and response
details. Up to ten `custom_sections` may be appended; each section has a title
of 1–80 characters and Markdown content of at most 4096 characters. HTML is
escaped, and links may use HTTPS or relative paths.

Branding and SEO settings belong in `yaml.d/branding.yml`, not `server.yml`.
The file is a normal config drop-in:

```yaml
server:
  branding:
    title: casman
    tagline: Universal Man Pages
    description: Man pages from BSD, macOS, Linux, and more
  seo:
    keywords: [manuals, man pages]
    author: casapps
    og_image: ""
    twitter_handle: ""
    title_separator: " - "
    meta_tags: {}
    canonical: ""
    noindex: false
    opengraph: {enabled: true, type: website, site_name: ""}
    twitter: {enabled: true, card: summary_large_image, creator: ""}
    verification: {google: "", bing: "", yandex: "", baidu: "", pinterest: "", facebook: "", custom: []}
```

`server.branding` controls the visible title, tagline, and page description.
`server.seo` controls keywords, author, the canonical URL prefix, OpenGraph
and Twitter cards, custom meta tags, noindex mode, and search-provider
verification tags. Unknown or unsafe meta names and invalid verification
values are ignored; verification content is HTML-escaped. Unlisted and
private pages default to `noindex,nofollow`, while the public home, browse,
search, page, and informational routes are explicitly indexable. Setting
`server.seo.noindex: true` marks every HTML page noindex and emits the matching
`X-Robots-Tag` header.

## Secrets

On first startup, the server creates `{config_dir}/secrets.yml` with an
operator token (`server.token`, `adm_` prefix) and an AES-256 encryption key
(`server.security.encryption_key`). The file is written atomically with mode
`0600`; keep it in the deployment's private configuration storage. The
operator token is printed once to the first-start console output. Later starts
show only its masked suffix. `GET /api/v1/server/auth/check` validates a
presented Bearer token and reports its kind without returning the credential.

The former `{config_dir}/security/master.key` file is migrated into
`secrets.yml` on startup so existing encrypted data remains readable.

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
- DNS-01 accepts the provider name in `server.ssl.letsencrypt.dns_provider`
  and the provider's flat credential map in
  `server.ssl.letsencrypt.dns_credentials` in `secrets.yml`. Credential values
  are encrypted at rest and are never logged.
- A daily scheduler task (03:00) parses each cached cert's `NotAfter` and
  re-provisions any cert with less than 30 days of life remaining. Static
  (non-ACME) certs are inspected too so operators get an early-warning
  log entry before expiry.
- HTTP-01 is answered on the HTTP listener and TLS-ALPN-01 on the HTTPS
  listener. Set `server.port` to a comma-separated pair such as `"80,443"`
  when both listeners are needed.

## Environment Variables

Directory variables are applied on every startup and use flag, environment,
then platform-default precedence:

| Variable | Flag | Purpose |
|----------|------|---------|
| `CONFIG_DIR` | `--config` | Configuration files |
| `DATA_DIR` | `--data` | Application and database data |
| `LOG_DIR` | `--log` | Log files |
| `BACKUP_DIR` | `--backup` | Backup files |
| `CACHE_DIR` | `--cache` | Cache files |
| `DATABASE_DIR` | — | SQLite database directory |

`DATABASE_URL`, `DATABASE_DRIVER`, and `CACHE_URL` are first-run setup inputs.
The saved file value wins on later starts; edit the files to change a saved
connection. `DATABASE_DRIVER` is used when the first-run database URL does not
select a driver. `APPLICATION_NAME` and `APPLICATION_TAGLINE` seed the branding
area file only on first run. `PORT` and `LISTEN` are read at every startup and
take precedence over saved port and address values unless their CLI flags are
set.

Other supported process settings include:

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
| `--mode` | production/development | production |
| `--config` | Config directory | OS-specific |
| `--data` | Data directory | OS-specific |
| `--debug` | Enable debug mode | false |

## Optional features

The server configuration also supports metrics, GeoIP, backup, and Tor
settings. These sections are optional; Tor startup is attempted automatically
when the `tor` executable is available on `PATH`. See the comments and defaults
in `src/config/config.go`. There is no admin web UI.

### Metrics

Metrics are served internally at `/server/metrics` and `/server/metrics/{service}`.
The root `/metrics` aliases are enabled by default and can be disabled with
`server.metrics.root.enabled: false`. Prometheus, Grafana, and Loki each use a
separate Bearer token, generated into `{config_dir}/secrets.yml` on startup.
Keep those credentials private and restrict metrics paths to trusted monitoring
clients at the firewall or reverse proxy. `auth.allow_unauthenticated` is only
for firewalled deployments.

```yaml
server:
  metrics:
    enabled: true
    root:
      enabled: true
    auth:
      allow_unauthenticated: false
    include_system: true
    include_runtime: true
    loki:
      max_entries: 1000
      max_age: 1h
```

The Loki endpoint streams recent structured HTTP request events and retains at
most `max_entries` events younger than `max_age`. Events include the normalized
route, method, status, duration, and byte counts. Request headers, query
strings, and bodies are not recorded; credential-like path values are redacted.

When enabled, a host-resource collector samples CPU, memory, and the filesystem
containing the configured data directory every 30 seconds. Runtime metrics use
the Go Prometheus collector. Scheduler metrics report run status, duration,
currently running tasks, and last-run timestamps. Business metrics expose the
catalog page count and successful man-page API requests. These values are also
included in the generated Grafana dashboard.

### CORS

`server.cors.allowed_origins` defaults to `[*]`. A wildcard never allows
credentials. For browser requests that need credentials, list the exact
origins and set `allow_credentials: true`. Use `allowed_origins: [""]` to
disable cross-origin headers. `max_age` defaults to 86400 seconds.

```yaml
server:
  cors:
    allowed_origins:
      - https://app.example.com
    allow_credentials: true
    max_age: 86400
```
