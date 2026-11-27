# Docker Rules (PART 26)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never add a Dockerfile USER directive or create the service account in the image.
- Never move the application Dockerfile out of docker/.
- Never make the entrypoint perform setup that the binary owns.

## CRITICAL - ALWAYS DO
- Start the container as root so the binary can prepare paths, bind ports, and drop privileges.
- Keep the entrypoint minimal and forward signals to the server.
- Keep image labels, health checks, volume paths, and documented ports aligned with runtime behavior.

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|---|---|---|
| What governs this area? | The server binary handles initialization; the container entrypoint only supplies environment and starts it. | AI.md PART 26 |

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
For complete details, see AI.md AI.md PART 26.
