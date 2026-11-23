# AI.md Alignment Backlog

The work is grouped by dependency: implement server preference and content
behavior before updating its browser forms; complete route and static-file
coverage before synchronizing the sitemap and user documentation.

## High (P1)

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
