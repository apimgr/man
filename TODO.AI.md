# AI.md Alignment Backlog

The work is grouped by dependency: implement server preference and content
behavior before updating its browser forms; complete route and static-file
coverage before synchronizing the sitemap and user documentation.

## High (P1)

- [ ] Complete the PART 20 metrics contract. This pass adds `/server/metrics`
  and service subroutes, API/root aliases, independent `mtr_` credentials in
  `secrets.yml`, constant-time Bearer checks, and initial Grafana JSON and
  Loki-compatible output. HTTP request counts, status, latency, body sizes,
  and active-request gauges now record through the outer middleware while
  health probes and metrics scrapes are excluded. Still missing: bounded
  structured log streaming with credential redaction, a complete dashboard
  covering every exported required metric, full system metric collection,
  configurable `loki.max_entries` and
  `loki.max_age`, verified exclusion from OpenAPI and public discovery, and
  exact endpoint/auth integration
  coverage. Relevant files: `src/metrics/`, `src/config/`, `src/secret/`,
  `src/server/`, `src/swagger/`, `docs/`.

- [ ] Complete validation and parity review of the required five-provider CI/CD
  set. GitHub, GitLab, Gitea, Forgejo, and Jenkins configurations now exist;
  verify each provider's build/test, security, release, and coverage behavior
  against AI.md, then confirm workflows on their respective hosted/self-hosted
  runners. Keep provider-specific limitations documented and resolve any drift.
- [x] Add the standard versioned and compatibility API aliases for Swagger and
  GraphQL, and keep OpenAPI, route tests, and the API guide synchronized.
  Implemented at `/api/v1/server/{swagger,graphql}` and `/api/{swagger,graphql}`;
  OpenAPI, route tests, and `docs/api.md` list the same routes.
- [ ] Finish verification of the AI.md contact and security-report flows.
  `/server/contact` now renders configured abuse, DMCA, privacy, and security
  forms with field descriptors, CAPTCHA/form tokens, validation, antispam,
  two-step confirmation, encrypted security-report storage, and API mirrors.
  `/server/security` and `security.txt` now render configured reporting
  contacts and expiry. Verify encrypted delivery/storage, PGP configuration,
  report status links, and the full documented failure matrix. Relevant files:
  `src/server/handler/contact*.go`, `src/server/securityreport/`,
  `src/config/`, `src/smtp/`.
- [ ] Implement the API text negotiation contract across every API route. The
  middleware now strips API `.txt` suffixes and serializes buffered JSON
  responses into text for `.txt`, `Accept: text/plain`, and detected HTTP
  tools; explicit JSON still wins over client detection. Remaining work is the
  canonical shared route registry with response-model metadata and primary
  fields, full consistency for every API error and binary route, and exact
  serializer behavior for all endpoint models. Relevant files:
  `src/server/middleware.go`, `src/server/api_text.go`, `src/server/server.go`,
  `src/server/handler/`, `src/swagger/`, `src/graphql/`.
- [ ] Implement the required always-on MCP Streamable HTTP endpoint at
  `/api/v1/server/mcp` with `/api/mcp` and `/mcp` aliases. It must enforce the
  AI.md protocol/header/Origin validation order, reuse API Bearer-token auth,
  expose only in-process tools backed by the route/OpenAPI registry, and keep
  token management and browser preferences out of MCP. A basic always-on
  handler, direct aliases, shared Bearer lookup, protocol/header/Origin checks,
  router dispatch, live-router/OpenAPI operation inventory, help text, unit
  coverage, and a built-server smoke for health plus operator/unauthenticated
  auth-check behavior now exist. API and security docs describe the transport
  and credentials. Remaining work: complete
  OpenAPI request schemas and meaningful operation descriptions; enforce the
  same API authorization, rate-limit and audit path with `source=mcp`; implement
  operator-token visibility and scope filtering; and pass the complete MCP
  acceptance matrix. Do not mark this complete until then.
  Relevant files: `src/server/`, `src/config/`, `src/swagger/`, `src/graphql/`,
  `src/server/template/`, `src/client/`.
- [ ] Complete catalog provenance and coverage from the IDEA.md source
  inventory. `make catalog` now reproducibly rebuilds the embedded SQLite
  file from the tracked
  pre-rendered platform directories, rejects implicit sample data, and checks
  SQLite/FTS integrity before replacing the asset. It sets deterministic
  man-page and configuration metadata timestamps with `SOURCE_DATE_EPOCH`
  (default `0`), since this corpus does not carry trustworthy per-page
  modification dates. Two consecutive builds of the tracked corpus produced
  byte-identical databases. `scripts/catalog-sources.tsv` now inventories
  IDEA.md source families, pins Linux man-pages 6.20, FreeBSD 15.1, OpenBSD
  8.0, NetBSD 11.1, and immutable latest-tag commits for selected Apple OSS
  repositories, and records authoritative checksums/license references where
  available. The matching download scripts consume those pins and verify BSD
  archive checksums. The current database is present and reproducibly rebuilt;
  the historical Git LFS object is not needed for that workflow. Remaining
  gaps: the checked-in corpus has no per-page license provenance and has
  lost original roff/mdoc for rendered pages; Apple package tags are not tied
  to one macOS point release; an attempted Apple `developer_cmds` package has
  no tagged release and is excluded; Debian and Alpine package branches remain
  unpinned; and IDEA.md sources such as GNU coreutils, util-linux, systemd,
  distribution/source mirrors, Void, DragonFly, and POSIX are not all locked,
  imported, rendered, or represented with per-page license metadata. Preserve
  source notices, add source version/license data to imported records, and
  complete the unresolved source inventory/import pipeline. Relevant files:
  `src/server/data/manpages.db`, `scripts/loader/`, `scripts/download_*.sh`,
  `scripts/catalog-sources.tsv`, `scripts/build_catalog.sh`.
- [ ] Complete guest preferences and the owner-token surface. Theme and
  language cookies, consent controls, reset, and stateless sync flows now
  exist with no-JavaScript forms. Owner-token storage, primary issue/restore,
  masked listing, one-page reveal, linked-token CRUD, rotation, expiry,
  revocation, cookie handling, and API mirrors are now implemented. Continue
  checking the full AI.md UI/API contract, strict rate limiting, audit details,
  MCP auth integration, CSRF, and preference consent/data validation before
  closing this item. Keep preferences browser-local and preserve the no-login
  product model. Relevant files: `src/server/server.go`, `src/server/handler/`,
  `src/server/store/`, `src/server/ownertoken/`, `src/server/template/`,
  `src/server/static/`.
- [x] Implement language-aware man page routes using stored translated rows;
  web and API routes return only the exact stored locale row and return 404
  when it is missing. The language inventory and preference selector derive
  available languages from stored row counts, with English as the empty-catalog
  fallback. Covered by `TestGetManPageLanguage` and handler route tests.
- [ ] Complete required static discovery behavior. Sitemap and robots output
  now apply the configured allow-minus-deny policy, and sitemap URLs include
  section, platform, and catalog pages; the sitemap index and numbered chunk
  handler are implemented. Remaining work: accurate `lastmod`, publication
  filtering, trusted request URL construction, and feed/sitemap validators and
  cache headers. Generate the `.well-known/llms.txt` endpoint inventory from
  one route registry while excluding methods and routes not meant for public
  discovery; the current route walk emits every method and route without
  visibility metadata.
  Relevant files: `src/server/server.go`, `src/server/handler/`.

## Medium (P2)

- [ ] Complete the required SEO metadata and validated verification-tag
  configuration, including canonical URLs, OpenGraph, Twitter cards, robots
  directives, and JSON-LD. Relevant files: `src/config/`,
  `src/server/template/`, `src/server/handler/`.
- [ ] Complete the frontend's no-JavaScript-first export and print flows and
  offline behavior for previously viewed man pages. Relevant files:
  `src/server/template/`, `src/server/static/`, `src/server/handler/`.
- [ ] Synchronize API and GraphQL reference output with every implemented
  endpoint and document the final browser and integration surfaces in `docs/`.
  Relevant files: `src/server/handler/handler.go`, `src/swagger/`,
  `src/graphql/`, `docs/`.
- [ ] Finish overlay-network compliance. Tor now uses the PART 32 torrc and
  control-port process with a dedicated PROXY-protocol backend listener, and
  Docker runtime health checks cover its hidden service. I2P now has a SAM
  client, eepsite key persistence, tunnel/status lifecycle, and configuration;
  finish checking the documented container/VM/host defaults, runtime health,
  and Docker/Incus behavior. Complete the operator-token route and
  authenticated CLI checks required by PART 28. Relevant files: `src/tor/`,
  `src/i2p/`, `src/server/`, `src/config/`, `tests/docker.sh`, `tests/incus.sh`.
- [ ] Complete PART 30 i18n and accessibility across the server, CLI, API,
  GraphQL, email, and documentation. The shared embedded package, six required
  locale files, request fallback selection, locale JSON endpoint, preference
  selector, and shared-layout translations now exist. Translate every
  remaining human-facing string, wire language selection through both
  binaries and all page/API responses, add plural/key/interpolation parity and
  accessibility checks, and synchronize the supported-language documentation.
  Relevant files: `src/common/i18n/`, `src/server/`, `src/client/`,
  `src/graphql/`, `src/config/`, `docs/`, `README.md`.
