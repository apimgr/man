# Integrations

## HTTP APIs

- REST API: `/api/v1/`; see [API Reference](api.md).
- OpenAPI JSON: `/api/v1/openapi`.
- Server-rendered API route documentation: `/openapi` (also `/swagger`).
- GraphQL query form: `/graphql`; JSON and form POST queries use `/graphql`.
- Atom updates: `/feed.xml`, with platform and section filters.
- RSS 2.0 updates: `/rss.xml`.
- JSON Feed: `/feed.json`.

## Browser and discovery

- `/manifest.json` and `/service-worker.js` provide the installable web app
  metadata, static asset caching, and an offline fallback page. HTML pages
  containing forms are not stored for offline use.
- `/robots.txt` identifies the sitemap at `/sitemap.xml`.
- `/sitemap-pages.xml`, `/sitemap-sections.xml`, and
  `/sitemap-platforms.xml` provide sitemap subdivisions.
- `/.well-known/security.txt` publishes vulnerability reporting information.
- `/.well-known/llms.txt` and `/llms.txt` list routes from the running router
  for machine clients.
- Feed and sitemap responses support `ETag`, `Last-Modified`, and conditional
  `304 Not Modified` responses.

## Optional network features

Tor and GeoIP support are configurable server features. Tor is disabled unless
enabled in `server.yml`; GeoIP is an optional risk signal and is not used as
the sole basis for access decisions. See [Configuration](configuration.md).

casman does not currently expose webhooks, OAuth/OIDC, federation, or native
application association endpoints.
