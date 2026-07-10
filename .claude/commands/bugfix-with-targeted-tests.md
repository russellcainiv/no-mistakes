---
name: bugfix-with-targeted-tests
description: Workflow command scaffold for bugfix-with-targeted-tests in no-mistakes.
allowed_tools: ["Bash", "Read", "Write", "Grep", "Glob"]
---

# /bugfix-with-targeted-tests

Use this workflow when working on **bugfix-with-targeted-tests** in `no-mistakes`.

## Goal

Fixes a bug or regression, often with targeted updates to both implementation and related tests.

## Common Files

- `internal/**/*.go`
- `internal/**/*_test.go`

## Suggested Sequence

1. Understand the current state and failure mode before editing.
2. Make the smallest coherent change that satisfies the workflow goal.
3. Run the most relevant verification for touched files.
4. Summarize what changed and what still needs review.

## Typical Commit Signals

- Update implementation files to resolve the bug.
- Add or update tests in *_test.go files to cover the bug scenario.
- Sometimes update related logic in closely associated files.

## Notes

- Treat this as a scaffold, not a hard-coded script.
- Update the command if the workflow evolves materially.