---
name: feature-development-with-tests-and-docs
description: Workflow command scaffold for feature-development-with-tests-and-docs in no-mistakes.
allowed_tools: ["Bash", "Read", "Write", "Grep", "Glob"]
---

# /feature-development-with-tests-and-docs

Use this workflow when working on **feature-development-with-tests-and-docs** in `no-mistakes`.

## Goal

Implements a new feature or major capability, including code changes, tests, and documentation updates.

## Common Files

- `internal/**/*.go`
- `internal/**/*_test.go`
- `docs/**/*.md`
- `README.md`

## Suggested Sequence

1. Understand the current state and failure mode before editing.
2. Make the smallest coherent change that satisfies the workflow goal.
3. Run the most relevant verification for touched files.
4. Summarize what changed and what still needs review.

## Typical Commit Signals

- Implement the new feature across relevant internal source files.
- Add or update corresponding tests in *_test.go files.
- Update or create documentation files (README.md, docs/*.md) to describe the new feature.

## Notes

- Treat this as a scaffold, not a hard-coded script.
- Update the command if the workflow evolves materially.