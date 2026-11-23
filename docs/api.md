# API Reference

## REST API

Base path: `/api/v1/`. Read endpoints return JSON. The public man page
collection is read-only.

### Man Page Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/man/{name}` | GET | Get man page by name |
| `/api/v1/man/{section}/{name}` | GET | Get man page by section and name |
| `/api/v1/man/{os}/{section}/{name}` | GET | Get man page for specific OS |
| `/api/v1/man/{lang}/{os}/{section}/{name}` | GET | Get an exact stored translation; 404 if unavailable |
| `/api/v1/compare/{name}` | GET | Compare across platforms |
| `/api/v1/compare/{section}/{name}` | GET | Compare within a section |
| `/api/v1/tldr/{name}` | GET | Get a quick summary and examples |
| `/api/v1/whatis/{name}` | GET | One-line description |
| `/api/v1/apropos?q={term}` | GET | Search descriptions |
| `/api/v1/search?q={term}` | GET | Search man pages |
| `/api/v1/autocomplete?q={term}` | GET | Autocomplete suggestions |
| `/api/v1/suggest?q={term}` | GET | Related page suggestions |
| `/api/v1/export/formats` | GET | List supported export formats |
| `/api/v1/whatis/{name}` | GET | One-line description |
| `/api/v1/apropos?q={term}` | GET | Search descriptions |
| `/api/v1/tldr/{name}` | GET | Get quick summary and examples |
| `/api/v1/compare/{name}` | GET | Compare across platforms |
| `/api/v1/compare/{section}/{name}` | GET | Compare within a section |
| `/api/v1/export/formats` | GET | List available export formats |

### System Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/healthz` | GET | Health check |
| `/api/v1/stats` | GET | Database statistics |
| `/api/v1/sections` | GET | List sections |
| `/api/v1/platforms` | GET | List platforms |
| `/api/v1/languages` | GET | List language metadata |
| `/api/v1/popular` | GET | List popular pages |
| `/api/v1/openapi` | GET | OpenAPI document |

There is no `/api/v1/man/{lang}/{name}` route. Translated retrieval requires
language, platform, section, and page name so it can match one stored row.

### Search Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query |
| `section` | string | Filter by section (1-9, `n`, or `x`) |
| `platform` | string | Filter by platform |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 25) |

## Swagger UI

Interactive API documentation: [/openapi](/openapi). The machine-readable
OpenAPI document is available from `/api/v1/openapi`.

## GraphQL

GraphQL UI: [/graphql](/graphql). Send GraphQL POST requests to the same path.

### Example Query

```graphql
query {
  manpage(name: "ls", section: "1") {
    name
    section
    title
    synopsis
    description
  }
}
```

## Browser routes

The server renders the homepage, man pages, search results, browse pages,
cross-platform comparisons, and `whatis`/`apropos` lookups. Man pages also
support `.txt`, `.md`, `.json`, `.raw`, `.pdf`, and `.epub` suffixes. Bulk
exports are available at `/export/section/{section}.{format}` and
`/export/platform/{platform}.{format}`.

`/healthz` negotiates HTML, JSON, or plain text. `/feed.xml` is Atom,
`/rss.xml` is RSS 2.0, and `/feed.json` is JSON Feed; per-platform and
per-section Atom feeds are also available.

Public information pages are available under `/server/about`, `/server/help`,
`/server/privacy`, `/server/terms`, `/server/security`, and `/server/contact`.
Visitor display preferences are saved at `/server/preferences`. The server has
no admin UI or account settings pages.
