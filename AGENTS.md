# Project SPEC

Project: See IDEA.md project variables
Role: Efficient loader for AI.md

⚠️ **THIS FILE IS AUTO-LOADED EVERY CONVERSATION. FOLLOW IT EXACTLY.** ⚠️

## Source of truth

- Read `AI.md` and `IDEA.md` before project work. `AI.md` defines implementation requirements; never edit it.
- Follow the applicable files in `.claude/rules/` and load relevant project memory from `.claude/memory/MEMORY.md`.
- Keep code, tests, documentation, and generated references aligned with `AI.md` and `IDEA.md`.

## Working rules

- Preserve unrelated working-tree changes and read files before editing them.
- Do not guess project behavior, identity, or configuration. Surface unresolved questions to the coordinating session.
- Run required validation in the prescribed Docker or Incus environment; never build or run the project locally.
- Follow `.git/COMMIT_MESS` and `gitcommit --dir {project_dir} all` for commits and normal pushes.
- After pushing, check CI and resolve failures before reporting completion.

For complete requirements, read `AI.md` and `IDEA.md` and the applicable `.claude/rules/` files.
