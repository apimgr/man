# Service Rules (PART 23, 24)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never run the server permanently as root unless IDEA.md explicitly requires it.
- Never daemonize a service-manager start or an application container.
- Never share the server PID file with a supervisor PID file.

## CRITICAL - ALWAYS DO
- Prepare the fixed service identity and owned paths before dropping privileges.
- Bind privileged sockets before dropping privileges, then verify the drop.
- Keep platform service installation, lifecycle, and status behavior aligned with each init system.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Service installation is distinct from server startup; systemd/OpenRC host detection precedes container defaults. | AI.md PART 23, 24 |

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
For complete details, see AI.md AI.md PART 23, 24.
