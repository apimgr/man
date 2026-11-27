# API Rules (PART 13, 14, 15)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never return placeholder or 501 API routes where the spec requires an implemented endpoint.
- Never expose internal errors, credentials, or unversioned breaking responses.
- Never bind a second ACME challenge socket that conflicts with application listeners.

## CRITICAL - ALWAYS DO
- Keep health/version behavior and API versioning consistent with the spec.
- Validate request input, use documented status codes, and keep OpenAPI/GraphQL references synchronized.
- Use the specified TLS profile, listener modes, and in-process ACME challenge handling.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | API behavior is defined by IDEA.md and PART 14; health/versioning and TLS are governed by PARTs 13 and 15. | AI.md PART 13, 14, 15 |

## TERMINOLOGY

| Term | Meaning |
|---|---|
| AI.md | Complete implementation specification and source of truth | PART 0 |
| IDEA.md | Project-specific product behavior and identity | PART 0 |

## QUICK REFERENCE

- Read the cited AI.md PARTs before editing this area.
- Follow all referenced sections and return to the point where the reference was encountered.
- Update paired docs and tests when behavior changes.

---
For complete details, see AI.md AI.md PART 13, 14, 15.
