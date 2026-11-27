# Frontend Rules (PART 16)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never require JavaScript for core functionality or implement an SPA/client-side router.
- Never put business logic in JavaScript, inline CSS/JS, or leave placeholder pages.
- Never allow long tokens, IPv6, or onion addresses to overflow mobile layouts.

## CRITICAL - ALWAYS DO
- Render pages with server-side Go templates and progressive enhancement.
- Use mobile-first responsive CSS, WCAG 2.1 AA accessibility, and 44px minimum touch targets.
- Source about/help content from IDEA.md and keep every server page fully functional.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Server pages are rendered by Go templates; page content sources and route requirements are in PART 16. | AI.md PART 16 |

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
For complete details, see AI.md AI.md PART 16.
