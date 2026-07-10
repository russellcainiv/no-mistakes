---
name: documentation-sync-with-code-changes
description: Workflow command scaffold for documentation-sync-with-code-changes in no-mistakes.
allowed_tools: ["Bash", "Read", "Write", "Grep", "Glob"]
---

# /documentation-sync-with-code-changes

Use this workflow when working on **documentation-sync-with-code-changes** in `no-mistakes`.

## Goal

Synchronizes documentation with recent code changes, ensuring published docs match new features, fixes, or workflows.

## Common Files

- `docs/**/*.md`
- `docs/**/*.mjs`
- `README.md`

## Suggested Sequence

1. Understand the current state and failure mode before editing.
2. Make the smallest coherent change that satisfies the workflow goal.
3. Run the most relevant verification for touched files.
4. Summarize what changed and what still needs review.

## Typical Commit Signals

- Update or create relevant documentation files to describe new or changed features.
- Ensure documentation site builds cleanly and all anchor links resolve.
- Update provider tables, environment variable docs, and configuration references as needed.

## Notes

- Treat this as a scaffold, not a hard-coded script.
- Update the command if the workflow evolves materially.