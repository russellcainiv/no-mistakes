# Review model wiring & the multi-reviewer panel

The review step is the pipeline stage that reads your diff and produces findings
(bugs, risks, simplifications). By default a **single** agent performs it — the
same agent that drives the rest of the pipeline. This document covers two
things:

1. How to wire the reviewer to a **different model**.
2. How to run a **multi-reviewer panel** (e.g. two models reviewing the same
   diff, findings unioned).

---

## 1. Wiring the reviewer to a different model

The reviewer is just the configured agent, so you choose its model the same way
you choose any agent's model — in `~/.no-mistakes/config.yaml`:

- **`agent`** selects the agent backend (`claude`, `codex`, `opencode`, `pi`,
  `copilot`, or `acp:<target>`). Different backends are effectively different
  models.
- **`agent_path_override`** points an agent at a specific binary — this is how a
  wrapper that pins a model (e.g. a `claude-mm` shim that runs the Claude CLI
  against a MiniMax endpoint) is selected.
- **`agent_args_override`** appends CLI flags for a native agent, e.g. a
  `--model` flag for Claude or `-m` for Codex. Flags that no-mistakes manages
  internally (`-p`, `--output-format`, `--json-schema`, …) are rejected.

```yaml
agent: claude
agent_args_override:
  claude: ["--model", "claude-opus-4-8"]
```

This is **global** to the run — it changes the model for every agent-backed
step (review, test-fix, document, PR summary), not review alone.

---

## 2. The multi-reviewer panel

Set `review.reviewers` to a list of two or more reviewers and the review step
becomes a **panel**: every reviewer reviews the diff **independently and
concurrently**, and their findings are **unioned and de-duplicated**. A problem
that *either* reviewer catches surfaces at the gate. This trades tokens/latency
for coverage — two models rarely miss the same bug.

Each reviewer entry has:

| Field   | Required | Meaning |
|---------|----------|---------|
| `agent` | yes      | Agent type whose CLI protocol to speak (`claude`, `codex`, `opencode`, `pi`, `copilot`, `acp:<target>`). `auto` is **not** allowed — a panel needs deterministic, explicit agents. |
| `path`  | no       | Override the binary for this reviewer only. This is how two entries of the **same** agent type run on **different** models. |
| `args`  | no       | Override CLI flags for this reviewer only (same reserved-flag rules as `agent_args_override`). |
| `label` | no       | Display name in logs and finding attribution. Defaults to the binary basename when `path` is set, otherwise the agent name. |

### Example — Claude + claude-mm (Claude and a MiniMax-backed Claude)

Both entries are `agent: claude` (both speak the Claude CLI protocol), but the
second points at the `claude-mm` wrapper binary, which execs the Claude CLI
against MiniMax. The wrapper sets its own endpoint/model env, so only `path` is
needed.

```yaml
review:
  reviewers:
    - agent: claude
    - agent: claude
      path: /Users/you/.local/bin/claude-mm
      label: claude-mm
```

### Example — pi + claude (two different agents)

```yaml
review:
  reviewers:
    - agent: pi
    - agent: claude
```

### Example — same agent, two models via args

```yaml
review:
  reviewers:
    - agent: claude
      args: ["--model", "claude-opus-4-8"]
    - agent: claude
      args: ["--model", "claude-haiku-4-5-20251001"]
      label: claude-haiku
```

## Semantics

- **Union + de-dup.** Findings from all reviewers are combined. Two findings are
  considered the same when they share file, line, and description (case- and
  whitespace-insensitive); duplicates collapse to one.
- **Attribution.** Each finding is tagged with the reviewer's `label` in its
  `source` field (unless the agent already attributed it).
- **Risk escalation.** The panel's risk level is the **highest** any reviewer
  assigned — one high-risk reviewer escalates the whole panel.
- **Best effort.** A reviewer whose binary can't be constructed (missing
  wrapper, bad path) is **skipped with a warning** rather than failing the run.
  If *every* reviewer fails, the step errors. An empty/omitted `reviewers` list
  falls back to the single pipeline agent (the default behavior).
- **Streaming.** With one reviewer, its output streams live. With a panel,
  per-reviewer streams are suppressed to avoid interleaving; you get a
  start line and a finding-count line per reviewer instead.

## Scope

`review.reviewers` only changes the **review** step. Every other step (test,
document, PR, CI) still uses the single pipeline `agent`. The panel does not
change how findings are approved — the same gate/approval flow applies to the
unioned set.
