# Makefile Rules (PART 25)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never use Makefile development targets as CI workflow definitions.
- Never build project binaries with a host Go toolchain.

## CRITICAL - ALWAYS DO
- Run build, local, test, and release targets through the prescribed Docker environment.
- Keep generated artifacts in scoped temporary/build locations and clean only task-created artifacts.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | The Makefile is the local development interface; CI behavior is defined separately in PART 27. | AI.md PART 25 |

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
For complete details, see AI.md AI.md PART 25.
