# CI/CD Rules (PART 27)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never use unpinned third-party workflow actions or host Go setup in CI.
- Never combine local Makefile development targets with CI workflow behavior.
- Never remove a provider workflow or security gate unless the spec explicitly says so.

## CRITICAL - ALWAYS DO
- Run the required workflow jobs in the specified Go container and dependency order.
- Keep lint, tests, security scans, build dependencies, image scanning, and coverage enforcement aligned across providers.
- Validate workflow YAML and required permissions before completion.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | CI builds are provider-specific implementations of the shared requirements in PART 27. | AI.md PART 27 |

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
For complete details, see AI.md AI.md PART 27.
