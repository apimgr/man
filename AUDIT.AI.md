# Alignment Audit

This file records open findings from the current AI.md alignment review. The
implementation backlog remains in TODO.AI.md; AI.md is authoritative.

## Open findings

- **Configuration reload coverage:** notification, logging, security-header,
  and ACME hot reloads are incomplete. Graceful timeout/listener replacement,
  Tor restart behavior, and the custom image pipeline remain unverified.
- **CLI coverage:** documented configuration sections and precedence edge
  cases, GUI behavior, accessibility, and complete user documentation remain
  incomplete.
- **Metrics acceptance:** route visibility still relies on route-name
  filtering, and the full endpoint/authentication acceptance matrix is missing.
- **Rate-limit acceptance:** the complete endpoint and storage-failover matrix
  has not been verified.
- **Contact and security flows:** encrypted delivery/storage, PGP handling,
  report status links, and the documented failure matrix need verification.
- **API contract consistency:** text negotiation still lacks a canonical route
  registry, per-route response metadata, and complete error/binary serializer
  coverage.
- **MCP authorization:** rate-limit/audit integration, operator visibility,
  token scope filtering, complete OpenAPI request schemas, and the acceptance
  matrix remain incomplete.
- **Catalog provenance:** the imported corpus lacks per-page license/source
  metadata; source pins/imports are incomplete for the IDEA.md inventory.
- **Browser preferences and offline flows:** consent/data validation and the
  complete no-JavaScript export/print matrix remain unverified.
- **Discovery and SEO:** route-registry visibility/schema output, trustworthy
  static-page modification dates, configured branding assets, and a complete
  server-page SEO inventory remain incomplete.
- **Overlay networking:** container/VM/host defaults, runtime health, Docker
  and Incus behavior, operator-token routes, and authenticated CLI checks need
  end-to-end verification.
- **Localization and accessibility:** human-facing strings remain untranslated
  across server, CLI, API, GraphQL, email, and docs; parity and accessibility
  checks are incomplete.
- **Provider CI parity:** configuration files exist for five providers, but
  parity and execution on the target provider runners have not been confirmed.

These findings are not closed by unit tests or successful builds alone. Close
each against the corresponding AI.md acceptance criteria and update TODO.AI.md
when the implementation status changes.
