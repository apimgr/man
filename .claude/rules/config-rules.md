# Configuration Rules (PART 5, 6, 12)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never add a runtime config mutation API or admin configuration UI.
- Never use generic PROJECT_* environment overrides for saved settings.
- Never let invalid configuration terminate a server that can still serve requests.

## CRITICAL - ALWAYS DO
- Keep operator settings file-backed in server.yml, ordered yaml.d drop-ins, and secrets.yml.
- Apply defaults, merge maps recursively, replace lists/scalars as whole values, and validate invalid values back to defaults with one warning.
- Keep restart-only settings and runtime reload behavior aligned with PART 12.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Configuration precedence and file ownership are defined by PART 5; application mode by PART 6; server settings by PART 12. | AI.md PART 5, 6, 12 |

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
For complete details, see AI.md AI.md PART 5, 6, 12.
