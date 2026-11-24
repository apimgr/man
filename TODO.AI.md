# AI.md Alignment Backlog

The work is grouped by dependency: implement server preference and content
behavior before updating its browser forms; complete route and static-file
coverage before synchronizing the sitemap and user documentation.

## High (P1)

- [ ] Add the standard versioned and compatibility API aliases for Swagger and
  GraphQL, and keep OpenAPI, route tests, and the API guide synchronized.
  Relevant files: `src/server/server.go`, `src/swagger/`, `src/server/server_extra_test.go`,
  `docs/api.md`.
- [ ] Implement the API text negotiation contract across every API route. The
  router does not strip `.txt` suffixes, API requests always negotiate JSON,
  and handlers serialize JSON regardless of the API text preference. Add the
  shared serializer and verify `.txt`, `Accept: text/plain`, explicit JSON,
  errors, and CLI defaults. Relevant files: `src/server/middleware.go`,
  `src/server/server.go`, `src/server/handler/`.
- [ ] Restore or reproducibly regenerate the embedded man-page catalog.
  `src/server/data/manpages.db` is a 134-byte Git blob containing a pointer to a
  241,754,112-byte object that is missing from the GitHub LFS store. There is no
  complete catalog build manifest or orchestrator: `scripts/loader --download`
  fetches a limited handpicked man7 set, its no-input mode loads samples, and
  the archive download scripts do not build the embedded database. Define and
  implement the supported source versions, metadata/licensing rules, complete
  import pipeline, output path, and integrity checks. Until then, page handlers
  and catalog APIs cannot serve the advertised content. Relevant files:
  `src/server/data/manpages.db`, `scripts/loader/`, `scripts/download_*.sh`.
- [ ] Complete guest preferences with cookie-backed language and project
  settings, reset and sync flows, and the credential-gated owner-token page.
  The theme preference POST exists; add the AI.md consent controls and make
  every preference form fully CSRF-verified and no-JavaScript-first. Preserve
  the product's no-account model. Relevant files: `src/server/server.go`,
  `src/server/handler/`, `src/server/template/`, `src/server/static/`.
- [ ] Implement language-aware man page routes using stored translated rows;
  stop advertising language coverage that is not backed by available content.
  Relevant files: `src/server/handler/`, `src/server/store/`,
  `src/server/model/`, `src/server/template/`.
- [ ] Complete required static discovery behavior: public-only sitemap and
  robots agreement, trusted request URL construction, and feed/sitemap
  validators and cache headers; generate the `.well-known/llms.txt` endpoint
  inventory from the registered routes rather than a separate list.
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
- [ ] Replace the Tor hidden-service setup with the PART 32 torrc/control-port
  integration and dedicated PROXY-protocol backend listener; the current Bine
  `Listen` implementation does not meet that requirement. Add the required
  Tor and I2P runtime checks to `tests/docker.sh` and complete operator-token
  route and authenticated CLI checks required by PART 28.
  Relevant files: `src/tor/`, `src/server/`, `tests/docker.sh`.
