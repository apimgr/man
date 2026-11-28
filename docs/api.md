# API Reference

## REST API

Base path: `/api/v1` by default (without a trailing slash). Set
`server.api_version` in the server's `server.yml` to select another version
prefix; routes, OpenAPI, MCP, and discovery use that configured value. The CLI
selects the matching client prefix in its own `cli.yml`. The public man-page
collection is read-only. JSON is the default API representation; clients
may request the shared plain-text representation as described below.

### Response Formats

API responses are JSON by default. Append `.txt` to an API route or send
`Accept: text/plain` to request the shared plain-text representation. The
`.txt` suffix takes precedence over `Accept`; otherwise, an explicit
`Accept: application/json` selects JSON. Non-interactive clients whose `Accept`
is absent or `*/*` receive text. Text objects use an `OK:` line followed by
`key: value` fields, lists identify the item count, and errors use
`ERROR: CODE: message`. Binary responses retain their original media type and
bytes.

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
| `/api/v1/server/healthz` | GET | Canonical versioned health API |
| `/api/healthz` | GET | Unversioned health API alias |
| `/api/v1/server/auth/check` | GET | Validate a Bearer token and return its kind; requires `Authorization: Bearer` |
| `/api/v1/stats` | GET | Database statistics |
| `/api/v1/sections` | GET | List sections |
| `/api/v1/platforms` | GET | List platforms |
| `/api/v1/languages` | GET | List language metadata |
| `/api/v1/popular` | GET | List popular pages |
| `/api/v1/server/metrics` | GET | Prometheus metrics; requires its service Bearer token unless unauthenticated access is explicitly configured |
| `/api/v1/server/metrics/grafana` | GET | Grafana dashboard JSON; requires the Grafana service Bearer token |
| `/api/v1/server/metrics/loki` | GET | Loki-compatible recent request events; requires the Loki service Bearer token |
| `/api/metrics` | GET | Unversioned Prometheus metrics alias |
| `/api/metrics/{prometheus,grafana,loki}` | GET | Unversioned metrics service aliases |
| `/api/v1/openapi` | GET | OpenAPI document |
| `/api/v1/server/swagger` | GET | OpenAPI document |
| `/api/v1/server/graphql` | GET, POST | GraphQL query endpoint |
| `/api/swagger` | GET | OpenAPI compatibility alias |
| `/api/graphql` | GET, POST | GraphQL compatibility alias |
| `/api/v1/server/about` | GET | Public service information |
| `/api/v1/server/privacy` | GET | Privacy information |
| `/api/v1/server/contact` | GET, POST | Report form descriptors and two-step abuse, DMCA, privacy, or security submission |
| `/api/v1/server/dpo` | GET | Public DPO contact; DPO email falls back to the general contact email, never the admin email |
| `/api/v1/server/help` | GET | Help information |
| `/api/v1/server/help/mcp` | GET | MCP integration help |
| `/api/v1/server/mcp` | POST | MCP Streamable HTTP endpoint; `/api/mcp` and `/mcp` are direct aliases |
| `/api/v1/server/terms` | GET | Terms of service |
| `/api/v1/server/security` | GET | Dynamic RFC 9116 contacts, expiry, PGP availability, policy, thanks, and report URLs |
| `/api/v1/server/security/policy` | GET | Public security disclosure policy |
| `/api/v1/server/security/thanks` | GET | Public researcher acknowledgments |
| `/api/v1/server/preferences` | GET, POST | Read or update guest theme and language preferences |
| `/api/v1/server/preferences/sync` | GET | Export guest preferences as a URL and short code |
| `/api/v1/server/preferences/sync/import` | GET | Import guest preferences from query parameters or a short code |
| `/api/v1/server/preferences/tokens` | GET | List masked owner and linked token metadata; requires the primary bearer token |
| `/api/v1/server/preferences/tokens/primary` | POST, DELETE | Issue an owner token publicly, or revoke the current owner with its primary bearer token |
| `/api/v1/server/preferences/tokens/primary/reveal` | POST | Reveal the encrypted primary token; requires the primary bearer token |
| `/api/v1/server/preferences/tokens/primary/rotate` | POST | Rotate the primary token without changing its owner; requires the primary bearer token |
| `/api/v1/server/preferences/tokens` | POST | Create a linked token; requires the primary bearer token |
| `/api/v1/server/preferences/tokens/{id}` | PATCH, DELETE | Update or revoke a linked token; requires the primary bearer token |
| `/api/v1/server/preferences/tokens/{id}/reveal` | POST | Reveal a linked token; requires the primary bearer token |
| `/api/v1/server/preferences/tokens/{id}/rotate` | POST | Rotate a linked token; requires the primary bearer token |
| `/api/v1` | GET | API service information and endpoint links; `/api/v1/` redirects here |

Security-report status pages have no API mirror. The reporter receives a
private status link at `/server/security/report/{tracking_id}`; its token can
be used once per UTC day and expires 30 days after the report is closed.

There is no `/api/v1/man/{lang}/{name}` route. Translated retrieval requires
language, platform, section, and page name so it can match one stored row.
Metrics credentials are separate service tokens generated into
`secrets.yml`; they are not owner or operator tokens. Metrics routes are
internal and do not appear in OpenAPI or MCP tool listings.

### MCP Streamable HTTP

Connect an MCP client to `/api/v1/server/mcp` (or either direct alias
`/api/mcp` and `/mcp`). The endpoint accepts JSON-RPC 2.0 POST requests using
protocol revision `2026-07-28`. Every request requires an existing API owner or
linked token in `Authorization: Bearer <token>`; cookies are ignored. Include
`MCP-Protocol-Version` matching `_meta.io.modelcontextprotocol/protocolVersion`
and `Mcp-Method` matching the JSON-RPC `method`. `tools/call` additionally
requires `Mcp-Name` matching `params.name`. Token issue, reveal, rotation, and
revocation are not available as MCP tools. Create a separate linked credential
for each client on [/server/preferences/tokens](/server/preferences/tokens).
The server currently returns a single JSON response per request and has no
session state.

### Search Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query |
| `section` | string | Filter by section (1-9, `n`, or `x`) |
| `platform` | string | Filter by platform |
| `page` | int | Page number (default: 1) |
| `limit` | positive int | Results per page (default: 20) |

## OpenAPI

Server-rendered Swagger UI: [/server/docs/swagger](/server/docs/swagger).
The machine-readable OpenAPI document is available from `/api/v1/openapi` and
`/api/v1/server/swagger`, with `/api/swagger` as the unversioned compatibility
alias.

## GraphQL

The server-rendered GraphQL query form is at
[/server/docs/graphql](/server/docs/graphql). It shares the application chrome and
provides a no-JavaScript link to the GraphQL endpoint. Requests use
`/api/v1/server/graphql`; `/api/graphql` is the unversioned compatibility alias.
POST accepts a JSON object with required `query` and optional `variables` and
`operationName`. GET uses the same names as query parameters; `variables` is a
JSON-encoded object. These fields are also published as the GraphQL operation's
OpenAPI and MCP input schemas.
The schema is read-only and exposes these query fields:

| Field | Arguments | Result |
|-------|-----------|--------|
| `version` | — | Server version string |
| `serverPage` | `path: String!` | Public server page content and sections |
| `manPage` | `name: String!`, `section: String`, `platform: String` | Man-page fields, including HTML, text, Markdown, and source |
| `search` | `query: String!`, optional `section`, `platform`, `page` (default `1`), `limit` (default `20`) | Search metadata and matching pages |
| `sections` | — | Catalog sections and counts |
| `platforms` | — | Catalog platforms and counts |
| `compare` | `name: String!`, optional `section` | Cross-platform page comparison |
| `stats` | — | Catalog totals and per-section/platform counts |
| `whatis` | `name: String!` | One-line descriptions |
| `apropos` | `query: String!` | Description search results |

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
  serverPage(path: "about") {
    title
    heading
    content
    format
  }
}
```

## Browser routes

The server renders the homepage, man pages, search results, browse pages,
cross-platform comparisons, `whatis`/`apropos` lookups, and owner-token
management at `/server/preferences/tokens`. Man pages also
support `.txt`, `.md`, `.json`, `.raw`, `.pdf`, `.epub`, and `.shtml`
(print-friendly HTML) suffixes. Bulk exports are available at
`/export/section/{section}.{format}` and
`/export/platform/{platform}.{format}`. Stored translations are rendered at
`/man/{lang}/{os}/{section}/{name}`.

`/healthz` negotiates HTML, JSON, or plain text. `/feed.xml` is Atom,
`/rss.xml` is RSS 2.0, and `/feed.json` is JSON Feed; per-platform and
per-section Atom feeds are also available.

Public information pages are available under `/server/about`, `/server/dpo`,
`/server/help`, `/server/privacy`, `/server/terms`, `/server/security`,
`/server/security/policy`, `/server/security/thanks`, and `/server/contact`.
The report form is intentionally unlinked outside the security overview and
`security.txt`. Its HTML responses are `no-store` and `noindex,nofollow`.
`GET /api/v1/server/contact` returns the form field descriptors, a one-use
simple CAPTCHA challenge, a signed form token, available report types, and
public security and abuse contact links. Supply a `type` query value to get
the fields for an abuse, DMCA, or privacy form. Submit the fields with
`step=submit` to receive a review token; repeat the fields with
`step=confirm`, that token, and `confirmed=true` to deliver the report.
Security submissions require a valid `security_id` from the instance's
security contact URL. The security overview is available from
`GET /api/v1/server/security` and `/.well-known/security.txt`. A successful
security submission returns its `tracking_id`; its emailed status link is a
one-use-per-UTC-day page at `/server/security/report/{tracking_id}` and has no
public API mirror. Source-code vulnerabilities should use the repository's
[GitHub private vulnerability reporting](https://github.com/apimgr/man/security/advisories/new),
not a deployed-instance form.
Visitor display preferences are saved at `/server/preferences`. The server has
no administration dashboard or user account system. The preferences page stores
theme and language choices, consent categories, and offers a reset action. The
separate `/server/preferences/tokens` page manages anonymous owner credentials;
the browser uses an HttpOnly primary-token cookie, while API routes require a
Bearer header and ignore cookies. Token list responses stay masked, and explicit
reveal responses are marked `no-store`. The separate
`/server/preferences/sync` page exports a URL and code and accepts an import;
its POST form is usable without JavaScript. The API mirrors are listed above.
The server also serves its embedded interface catalogue at
`/locales/{lang}.json`; supported UI locales are English, Spanish, French,
German, Italian, and Arabic. These locale files are separate from stored
man-page translations returned by `/api/v1/languages`.

### Owner Tokens

Owner tokens are anonymous credentials, not user accounts. `POST
/api/v1/server/preferences/tokens/primary` creates an owner and returns its
primary credential in `data.owner_token`. A valid owner or linked bearer on
that request returns `409`. All other token-management API routes require the
primary token as `Authorization: Bearer own_...`; linked tokens receive `403`,
and API routes ignore browser cookies. Token list results contain masked values
only. Explicit reveal, rotate, and creation responses carry the raw credential
in the JSON body and use `Cache-Control: no-store`.

Create a linked token with `POST /api/v1/server/preferences/tokens` and
`{"name":"mcp-client","scope":"read-write","expires":"30d"}`. Scope is
`read` or `read-write` (default); `global` is reserved for the primary token.
Read-only linked credentials see only GET tools over MCP. A token's scope cannot
be changed after creation. Supported expiries are
`never`, `7d`, `30d`, `90d`, and `1y`; a chosen UTC calendar date uses
`expires_on: "YYYY-MM-DD"`. The web token page is
`/server/preferences/tokens`; it uses the HttpOnly primary-token cookie and
CSRF-protected POST forms. API token routes have no unversioned aliases.
