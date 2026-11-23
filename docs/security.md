# Security

## Service model

casman serves a read-only public man page collection. It does not provide user
accounts, an admin web UI, page editing, or browser-based server settings.
Configure the server through `server.yml` and the command line. Do not expose
private configuration, backup, or data directories through the web server.

## Network protection

- Use the built-in TLS configuration or place casman behind a trusted TLS
  reverse proxy. TLS settings and ACME options are documented in
  [Configuration](configuration.md).
- The server applies request rate limits and security middleware. Keep the
  service and its embedded dependencies updated.
- `/metrics` is disabled unless enabled in configuration. Set its token before
  exposing the endpoint outside a trusted monitoring network.
- CORS is intended for public API reads; do not send credentials with wildcard
  origins.

## Reporting vulnerabilities

`/.well-known/security.txt` publishes the reporting contact derived from the
configured server hostname. Verify the contact is
reachable before deploying a public instance. Reports should not be sent in
public issues before a fix is available.

## Data and privacy

Bookmarks, browsing history, and theme selection are browser-side features;
they do not require an account. Server request logs and operational data are
managed by the deployment. Review the deployment's log retention and backup
settings before making the service available to users.
