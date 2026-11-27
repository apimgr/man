# Server Operations

casman is a read-only public man page service. It has no admin web UI, user
accounts, setup wizard, or admin API endpoints. Configure and operate
the server through `server.yml`, environment variables, and the `casman`
command line.
Versioned API examples use the default `/api/v1` prefix; `server.api_version`
in `server.yml` controls the canonical versioned routes.

Public information pages are available at `/server/about`, `/server/help`,
`/server/help/mcp`, `/server/privacy`, `/server/terms`, `/server/security`,
`/server/security/policy`, `/server/security/thanks`, and `/server/contact`.
Security-report status pages use the private link sent to the reporter at
`/server/security/report/{tracking_id}`; the link token is rate-limited to one
view per UTC day and expires 30 days after the report is closed. Visitors can
save browser-only display and consent preferences at `/server/preferences`,
export or import them at `/server/preferences/sync`, and manage owner and
linked API tokens at `/server/preferences/tokens`. These pages do not change
server configuration or create user accounts.

## Public status and statistics

- `/server/healthz` serves HTML to browsers and supports JSON or plain text
  through content negotiation; `/healthz` is the root alias.
- `/api/v1/server/healthz` and `/api/healthz` return health data as JSON. The
  versioned `/api/v1/healthz` route remains available for compatibility.
- `/api/v1/stats` returns embedded database statistics.
- `/server/metrics` and `/metrics` serve Prometheus exposition. The
  `/server/metrics/{prometheus,grafana,loki}` paths, root aliases under
  `/metrics`, and API aliases at `/api/v1/server/metrics` and `/api/metrics`
  use separate Bearer tokens stored in `secrets.yml`.
  Restrict access to monitoring clients. The Loki endpoint returns bounded,
  structured HTTP request events with request headers, query strings, and
  bodies excluded and credential-like path values redacted. Prometheus also
  exports host-resource, scheduler, catalog-size, and man-page API request
  metrics; the generated Grafana dashboard includes panels for those families.

## Configuration and maintenance

See [Configuration](configuration.md) for server, database, TLS, metrics,
backup, GeoIP, and Tor settings. Use the documented CLI flags and maintenance
commands for operations; there is no browser-based administration or
configuration restore page.
