# Integrations

## HTTP APIs

The versioned API examples below use the default `/api/v1` prefix. Set
`server.api_version` in `server.yml` to use another version.

- REST API: `/api/v1/`; see [API Reference](api.md).
- OpenAPI JSON: `/api/v1/openapi`.
- Server-rendered API documentation: `/server/docs/swagger`.
- GraphQL query form: `/server/docs/graphql`; queries use `/api/v1/server/graphql`.
- Atom updates: `/feed.xml`, with platform and section filters.
- RSS 2.0 updates: `/rss.xml`.
- JSON Feed: `/feed.json`.

## MCP Streamable HTTP

The server exposes JSON-RPC tools at `/api/v1/server/mcp`, with direct aliases
at `/api/mcp` and `/mcp`. Send `POST` requests with a Bearer token and protocol
revision `2026-07-28` in both `MCP-Protocol-Version` and
`_meta.io.modelcontextprotocol/protocolVersion`; `Mcp-Method` must match the
JSON-RPC method. Tool calls also send `Mcp-Name` matching `params.name`.
Requests are stateless and return one JSON response. Cookies are not used for
MCP authentication. See [API Reference](api.md#mcp-streamable-http) for the
request contract and token-management exclusions.

## Browser and discovery

- `/manifest.json` and `/service-worker.js` provide the installable web app
  metadata and static asset caching. Previously viewed public man pages and
  public information pages can be used offline when requested without a
  visitor cookie or Authorization header. The worker synthesizes a translated
  offline page when no eligible cached page is available; there is no static
  offline HTML page, and API responses are never cached. The worker caps its
  versioned cache at 50 MiB, expires cached static assets after seven days, and
  checks available browser storage quota before writing cache entries.
- `/robots.txt` identifies the sitemap at `/sitemap.xml`.
- `/sitemap.xml` lists public pages and catalog entries, honoring the configured
  `web.robots.allow` and `web.robots.deny` paths. Catalogs above 50,000 URLs
  use `/sitemap-1.xml`, `/sitemap-2.xml`, and subsequent numbered files.
- `/sitemap-pages.xml`, `/sitemap-sections.xml`, and
  `/sitemap-platforms.xml` remain available as focused sitemap views.
- `/.well-known/security.txt` publishes vulnerability reporting information.
- `/.well-known/llms.txt` and `/llms.txt` list live API operations with their
  OpenAPI summaries and request-specific API/MCP URLs for machine clients.
- Feed and sitemap responses support `ETag`, `Last-Modified`, and conditional
  `304 Not Modified` responses.

## Optional network features

Tor hidden-service startup is attempted automatically when a `tor` executable
is available on `PATH`; its data directory, virtual port, and bootstrap timeout
can be configured in `server.yml`. GeoIP is an optional risk signal and is not
used as the sole basis for access decisions. See
[Configuration](configuration.md).

casman does not currently expose webhooks, OAuth/OIDC, federation, or native
application association endpoints.
