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
| `/api/v1/server/swagger` | GET | OpenAPI document alias |
| `/api/v1/server/graphql` | GET, POST | GraphQL query endpoint |
| `/api/swagger` | GET | OpenAPI compatibility alias |
| `/api/graphql` | GET, POST | GraphQL compatibility alias |
| `/api/v1/server/security/policy` | GET | Public security disclosure policy |
| `/api/v1/server/preferences` | GET, POST | Read or update guest theme and language preferences |
| `/api/v1/server/preferences/sync` | GET | Export guest preferences as a URL and short code |
| `/api/v1/server/preferences/sync/import` | GET | Import guest preferences from query parameters or a short code |
| `/api/v1/` | GET | API service information and endpoint links |

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

## OpenAPI

Server-rendered Swagger UI: [/server/docs/swagger](/server/docs/swagger),
with `/openapi` and `/swagger` aliases. The machine-readable OpenAPI document
is available from `/api/v1/openapi`, `/openapi.json`, and `/swagger.json`.

## GraphQL

The server-rendered GraphiQL page is at
[/server/docs/graphql](/server/docs/graphql), with `/graphql` and `/graphiql`
aliases. GraphQL requests can also be sent as JSON POSTs to `/graphql`.

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
support `.txt`, `.md`, `.json`, `.raw`, `.pdf`, `.epub`, and `.shtml`
(print-friendly HTML) suffixes. Bulk exports are available at
`/export/section/{section}.{format}` and
`/export/platform/{platform}.{format}`. Stored translations are rendered at
`/man/{lang}/{os}/{section}/{name}`.

`/healthz` negotiates HTML, JSON, or plain text. `/feed.xml` is Atom,
`/rss.xml` is RSS 2.0, and `/feed.json` is JSON Feed; per-platform and
per-section Atom feeds are also available.

Public information pages are available under `/server/about`, `/server/help`,
`/server/privacy`, `/server/terms`, `/server/security`,
`/server/security/policy`, and `/server/contact`.
Visitor display preferences are saved at `/server/preferences`. The server has
no admin UI or account settings pages. The preferences page stores theme and
language choices, consent categories, and offers a reset action. The separate
`/server/preferences/sync` page exports a URL and code and accepts an import;
its POST form is usable without JavaScript. The API mirrors are listed above.
