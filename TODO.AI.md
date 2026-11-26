# AI.md Alignment Backlog

The work is grouped by dependency: implement server preference and content
behavior before updating its browser forms; complete route and static-file
coverage before synchronizing the sitemap and user documentation.

## High (P1)

- [ ] Implement PART 5 area-file ownership, precedence, generated-file split,
  legacy `server.yaml` migration, automatic splitting of area settings from
  `server.yml`, path-safe drop-in loading, duplicate/null-key recovery, source
  attribution, permission enforcement, and hot reload. Current `Load` only
  merges parseable `yaml.d` files into the config struct and validates the
  final merged values; it does not split legacy sections or keep their source
  file attribution. `Config.Save` serializes every loaded section back into
  `server.yml`. Relevant files: `src/config/`, server startup/reload, and
  configuration docs. Required by AI.md PART 5 → “Drop-in Config Directory,”
  “Automatic Migration,” and “Area Files and Automatic Migration.”

- [ ] Complete the thin, testable server entrypoint required by PART 25.
  Build metadata is in `src/version.go`, and server startup/shutdown now run
  through `src/app.Run(ctx, cfg)` and `Server.RunContext` with an injectable
  listener. A runner test makes a real health request, cancels the context,
  and checks clean shutdown. `src/main.go` now contains only the executable
  bootstrap; flag parsing and actions are in the testable `src/cli` package,
  and build metadata is copied into runtime config. The full test gate,
  cross-platform build, browser E2E suite, and phase-two compiled-binary
  validation all pass. This implementation item is complete and will be
  removed with the final aligned change.
- [ ] Complete the PART 33/34 CLI configuration contract. Server and token
  resolution now reads `cli.yml`, supports `--config`, validates URLs, keeps
  config files private, honors explicit flags and environment settings, and
  avoids an implicit localhost server. API version, timeout, retry count and
  delay, `--token-file`, and config token-file loading now reach both the
  command runner and TUI; the
  CLI reference documents these settings. Remaining work includes the rest of
  the documented config sections and flag/environment precedence, config
  path/profile edge cases, accessibility and
  GUI coverage, and complete CLI documentation. Output format and pager
  preferences now honor CLI flags, environment variables, and `cli.yml`;
  JSON, YAML, plain, table, and CSV serializers are implemented. Color
  preferences honor `--color`, `output.color`, `NO_COLOR`, and
  `TERM=dumb`; quiet and verbose preferences honor flags, environment values,
  and `cli.yml`. Table headings receive color, quiet mode omits nonessential
  search/man-page framing, and verbose mode reports the request server on
  stderr. TUI theme, mouse, and Unicode settings now configure the Bubble Tea
  interface; its command menu covers all eight commands exposed by `--help`
  with man-page, search, whatis, apropos, statistics, section, platform, and
  health views. Response caching now honors `cache.enabled`, `cache.ttl`, and
  `cache.max_size`, scopes entries by server and token, and keeps private files
  under the user cache directory. The logger now honors its configured level,
  path, size cap, and retention count, writes plain text to private user-scoped
  files, and records command failures without credentials. New config files
  contain documented defaults.
  Interactive first-run setup now prompts for a server and
  optional owner token in Bubble Tea, checks reachability/authentication,
  preserves existing valid values, and saves the initial config only after
  successful checks. CLI errors now map to the specified usage, validation,
  not-found, connection, auth/permission, and config exit codes; `health`
  returns failure for a degraded server. Relevant files: `src/client/`,
  `docs/`.
- [ ] Complete the PART 20 metrics contract. `/server/metrics` and its service
  subroutes, API/root aliases, independent `mtr_` credentials in `secrets.yml`,
  constant-time Bearer checks, and initial Grafana JSON and Loki-compatible
  output exist. HTTP request counts, status, latency, body sizes, and
  active-request gauges record through the outer middleware while health
  probes and metrics scrapes are excluded. Loki now retains bounded structured
  HTTP events using `loki.max_entries` and `loki.max_age`; it records normalized
  route, method, status, duration, and byte counts without headers, query
  strings, or bodies, and redacts credential-like path values. Scheduler task
  lifecycle metrics, catalog-size and man-page API business metrics, matching
  Grafana panels, and a periodic CPU/memory/data-volume collector are now
  implemented. Metrics routes are excluded from OpenAPI, public discovery,
  and MCP; tests cover service-specific credentials across all metrics aliases
  and preserve Grafana/Loki JSON media types. Remaining work: replace route-name
  filtering with route-registry visibility metadata and complete the full
  endpoint/auth acceptance matrix. Relevant files:
  `src/metrics/`,
  `src/config/`, `src/secret/`,
  `src/server/`, `src/swagger/`, `docs/`.

- [ ] Implement the complete `server.rate_limit` configuration and enforcement
  from PART 12. Read/write limits use per-IP or hashed Bearer-token buckets;
  health and strict token-action windows, the per-IP global burst ceiling,
  and invalid-Bearer failure lockout are configured and enforced. API 429
  responses include the documented error envelope and `Retry-After`; lockout
  uses the same 429 response until the sliding failure count drops below its
  limit. SQLite stores atomic current/previous fixed-window counters and
  applies the documented weighted sliding-window calculation; counters survive
  process restarts. Defaults, validation, route classification, owner
  bucketing, global ceilings, and lockout have focused tests. `server.cache`
  now supports none, memory, Valkey, and Redis backends; URL and TLS/CA/client
  certificate configuration, pools/timeouts, generic cache operations,
  health/metrics, and atomic Redis sliding counters are wired. Successful
  Redis counter operations also persist a SQLite shadow, so cache loss and
  restart do not discard rate-limit history; cache errors use the persistent
  store before the in-process fallback. A live Valkey integration test verifies
  shared Lua counters, strict lockout checks, and continued denial after the
  cache is flushed. Remaining work is the full endpoint/failover acceptance
  matrix. Relevant files: `src/config/`,
  `src/server/middleware.go`, `src/server/server.go`, `src/server/store/`,
  `src/cache/`, and `docs/`.

- [ ] Complete validation and parity review of the required five-provider CI/CD
  set. GitHub, GitLab, Gitea, Forgejo, and Jenkins configurations now exist;
  verify each provider's build/test, security, release, and coverage behavior
  against AI.md, then confirm workflows on their respective hosted/self-hosted
  runners. Keep provider-specific limitations documented and resolve any drift.
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
  checking the full AI.md UI/API contract, audit details,
  MCP auth integration, CSRF, and preference consent/data validation before
  closing this item. Strict token actions now use configurable
  `server.rate_limit.strict` limits (10 requests per 900 seconds by default),
  bucket valid credentials by owner and otherwise use the resolved client
  address, and count invalid attempts. API throttles return the documented
  `RATE_LIMITED` JSON envelope with `Retry-After`. Focused tests cover the
  strict route matrix, owner bucketing, API error output, and config defaults
  and validation. Keep preferences browser-local and preserve the no-login
  product model. Relevant files: `src/server/server.go`, `src/server/handler/`,
  `src/server/store/`, `src/server/ownertoken/`, `src/server/template/`,
  `src/server/static/`.
- [ ] Complete required static discovery behavior. Sitemap and robots output
  now apply the configured allow-minus-deny policy, and sitemap URLs include
  section, platform, and catalog pages; the sitemap index and numbered chunk
  handler are implemented. Feeds use catalog creation dates and exclude rows
  without trustworthy creation times; sitemap catalog URLs include
  `lastmod` only when the stored timestamp is trustworthy. Shared URL
  construction now applies the trusted proxy and overlay-network rules to
  feeds, sitemaps, and robots output. Remaining work: trustworthy modification
  dates for static page content. `.well-known/llms.txt` now uses the live
  OpenAPI operation inventory for summaries, auth labels, parameter/response
  details, and request-resolved API/MCP/documentation URLs; `web.llms` controls
  endpoint details and validated custom sections. Remaining work: use the
  shared route registry's `Listed` visibility metadata instead of deriving
  visibility from OpenAPI/path rules, and emit complete request/response
  schemas from the route models.
  Relevant files: `src/server/server.go`, `src/server/handler/`.

## Medium (P2)

- [ ] Complete the required SEO metadata and validated verification-tag
  configuration. Server-rendered canonical, OpenGraph, Twitter, JSON-LD,
  custom meta, provider verification, and fail-closed robots metadata now use
  `server.seo`; tests cover output, invalid settings, public/private routes,
  and noindex mode. Remaining: validate branding image inputs, honor configured
  favicon/logo assets, and review every server page against the SEO route
  inventory. Relevant files: `src/config/`, `src/server/template/`,
  `src/server/handler/`.
- [ ] Complete the frontend's no-JavaScript-first export and print flows and
  offline behavior for previously viewed man pages. The worker now caches
  only eligible public man/information pages, synthesizes its fallback from
  translated strings stored by the page, and never serves `/offline.html` or
  caches API responses. Its versioned cache is capped at 50 MiB, static assets
  expire after seven days, and writes check browser storage quota. Focused tests
  cover cache eligibility and assert the limit/expiry/quota policy is emitted.
  Remaining: audit every export and print flow in a no-JS browser and exercise
  the worker behavior in browser E2E tests. Relevant
  files: `src/server/template/`, `src/server/static/`, `src/server/handler/`.
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
