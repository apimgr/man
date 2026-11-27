# Project Rules (PART 2, 3, 4)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never add generated-by or AI authorship lines.
- Never place runtime data, build output, logs, or secrets in source-controlled paths.
- Never use non-portable paths or assume one operating system's directory layout.

## CRITICAL - ALWAYS DO
- Use the project license and attribution exactly as specified.
- Keep source, docs, and committed team configuration in their prescribed locations.
- Resolve platform paths from the startup context and cache them for the process lifetime.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Runtime data belongs in configured data/cache/log paths; source and docs belong in the repository. | AI.md PART 2, 3, 4 |

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
For complete details, see AI.md AI.md PART 2, 3, 4.
