# Server Operations

casman is a read-only public man page service. It has no admin web UI, user
accounts, setup wizard, or `/api/v1/admin/` endpoints. Configure and operate
the server through `server.yml`, environment variables, and the `casman`
command line.

Public information pages are available at `/server/about`, `/server/help`,
`/server/privacy`, `/server/terms`, `/server/security`, and `/server/contact`.
Visitor display preferences are saved at `/server/preferences`.

## Public status and statistics

- `/healthz` serves HTML to browsers and supports JSON or plain text through
  content negotiation.
- `/api/v1/healthz` returns health data as JSON.
- `/api/v1/stats` returns embedded database statistics.
- `/metrics` is the Prometheus endpoint. It is disabled unless enabled in
  configuration and can be protected with its configured token.

## Configuration and maintenance

See [Configuration](configuration.md) for server, database, TLS, metrics,
backup, GeoIP, and Tor settings. Use the documented CLI flags and maintenance
commands for operations; there is no browser-based settings or restore page.
