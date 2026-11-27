# Testing Rules (PART 28, 29, 30)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never build or test the project on the host when the spec requires Docker or Incus.
- Never skip required tests, coverage thresholds, documentation builds, or accessibility checks.
- Never claim a check passed unless its actual output was observed.

## CRITICAL - ALWAYS DO
- Use the prescribed Docker/Incus test hierarchy and keep test artifacts in project-scoped temporary storage.
- Maintain the required coverage floor and test supported OS, API routes, templates, security, and failure behavior.
- Build MkDocs documentation and preserve I18N, RTL, keyboard, and accessibility requirements.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Testing/development, ReadTheDocs, and I18N/accessibility requirements are all release gates. | AI.md PART 28, 29, 30 |

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
For complete details, see AI.md AI.md PART 28, 29, 30.
