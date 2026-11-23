# Integrations

## HTTP APIs

- REST API: `/api/v1/`; see [API Reference](api.md).
- OpenAPI JSON: `/api/v1/openapi`.
- Swagger UI: `/openapi` (also `/swagger`).
- GraphQL UI: `/graphql`; POST GraphQL requests to `/graphql`.
- Atom updates: `/feed.xml`, with platform and section filters.
- RSS 2.0 updates: `/rss.xml`.
- JSON Feed: `/feed.json`.

## Browser and discovery

- `/manifest.json` and `/service-worker.js` provide the installable web app
  metadata and offline cache support.
- `/robots.txt` identifies the sitemap at `/sitemap.xml`.
- `/sitemap-pages.xml`, `/sitemap-sections.xml`, and
  `/sitemap-platforms.xml` provide sitemap subdivisions.
- `/.well-known/security.txt` publishes vulnerability reporting information.
- `/.well-known/llms.txt` lists public API entry points for machine clients.

## Optional network features

Tor and GeoIP support are configurable server features. Tor is disabled unless
enabled in `server.yml`; GeoIP is an optional risk signal and is not used as
the sole basis for access decisions. See [Configuration](configuration.md).

casman does not currently expose webhooks, OAuth/OIDC, federation, or native
application association endpoints.
