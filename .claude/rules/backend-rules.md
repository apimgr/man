# Backend Rules (PART 9, 10, 11, 31)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never expose secrets in logs, errors, metrics, or public responses.
- Never make optional upstreams, datasets, or overlay services prevent the server from starting.
- Never store operator-editable configuration in the database.

## CRITICAL - ALWAYS DO
- Return actionable errors and degrade optional dependencies while keeping available routes online.
- Use the specified database schema, migrations, encryption, audit, and logging behavior.
- Run Tor/I2P as optional managed features according to their distinct enablement rules.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | The database stores application state; config files remain authoritative; Tor and I2P have separate defaults. | AI.md PART 9, 10, 11, 31 |

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
For complete details, see AI.md AI.md PART 9, 10, 11, 31.
