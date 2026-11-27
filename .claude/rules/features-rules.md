# Feature Rules (PART 17-22)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never require external cron for scheduled application tasks.
- Never mark optional datasets or third-party upstreams critical to basic service health.
- Never write backup data to a silent fallback directory when the configured destination fails.

## CRITICAL - ALWAYS DO
- Use the built-in scheduler with retries and task history.
- Treat email, GeoIP, metrics, backup, and update providers as configured optional integrations with clear failure reporting.
- Keep backup permissions, retention, and restore validation aligned with the spec.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | Optional integrations degrade independently; scheduler, backup, and update behavior are specified in PARTs 18, 21, and 22. | AI.md PART 17-22 |

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
For complete details, see AI.md AI.md PART 17-22.
