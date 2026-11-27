# Binary Rules (PART 7, 8, 32)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never enable CGO or build server/client binaries with host toolchains.
- Never hand-roll the server's primary flag parser contrary to the CLI rules.
- Never omit the required client binary or its supported platforms.

## CRITICAL - ALWAYS DO
- Build stripped, reproducible binaries with the required names and build metadata.
- Handle immediate-exit and service/maintenance commands before server startup.
- Keep daemon re-exec arguments free of the daemon flag and preserve the child marker.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Server and client are separate required binaries; command contracts and exit codes are specified in PART 8 and PART 32. | AI.md PART 7, 8, 32 |

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
For complete details, see AI.md AI.md PART 7, 8, 32.
