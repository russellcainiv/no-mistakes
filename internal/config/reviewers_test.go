package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func writeGlobalConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadGlobalReviewersClaudeAndClaudeMM(t *testing.T) {
	path := writeGlobalConfig(t, `
agent: claude
review:
  reviewers:
    - agent: claude
    - agent: claude
      path: /Users/russell/.local/bin/claude-mm
      label: claude-mm
`)
	cfg, err := LoadGlobal(path)
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if len(cfg.Reviewers) != 2 {
		t.Fatalf("want 2 reviewers, got %d: %+v", len(cfg.Reviewers), cfg.Reviewers)
	}
	if cfg.Reviewers[0].Agent != types.AgentClaude || cfg.Reviewers[0].Path != "" {
		t.Errorf("reviewer 0 = %+v, want default claude", cfg.Reviewers[0])
	}
	if cfg.Reviewers[1].Agent != types.AgentClaude ||
		cfg.Reviewers[1].Path != "/Users/russell/.local/bin/claude-mm" ||
		cfg.Reviewers[1].Label != "claude-mm" {
		t.Errorf("reviewer 1 = %+v, want claude-mm wrapper", cfg.Reviewers[1])
	}

	// Merge must carry reviewers through to the resolved Config.
	merged := Merge(cfg, &RepoConfig{})
	if len(merged.Reviewers) != 2 {
		t.Fatalf("merged reviewers = %d, want 2", len(merged.Reviewers))
	}
}

func TestLoadGlobalReviewersPiAndClaude(t *testing.T) {
	path := writeGlobalConfig(t, `
review:
  reviewers:
    - agent: pi
    - agent: claude
`)
	cfg, err := LoadGlobal(path)
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if len(cfg.Reviewers) != 2 || cfg.Reviewers[0].Agent != types.AgentPi || cfg.Reviewers[1].Agent != types.AgentClaude {
		t.Fatalf("unexpected reviewers: %+v", cfg.Reviewers)
	}
}

func TestLoadGlobalReviewersRejectAuto(t *testing.T) {
	path := writeGlobalConfig(t, `
review:
  reviewers:
    - agent: claude
    - agent: auto
`)
	if _, err := LoadGlobal(path); err == nil {
		t.Fatal("expected error for auto reviewer, got nil")
	}
}

func TestLoadGlobalReviewersRejectUnknown(t *testing.T) {
	path := writeGlobalConfig(t, `
review:
  reviewers:
    - agent: gpt5
`)
	if _, err := LoadGlobal(path); err == nil {
		t.Fatal("expected error for unknown reviewer, got nil")
	}
}

func TestLoadGlobalReviewersRejectReservedArg(t *testing.T) {
	// -p is a managed claude flag and must not be overridable per reviewer.
	path := writeGlobalConfig(t, `
review:
  reviewers:
    - agent: claude
      args: ["-p", "hello"]
`)
	if _, err := LoadGlobal(path); err == nil {
		t.Fatal("expected error for reserved reviewer arg, got nil")
	}
}

func TestLoadGlobalReviewersAllowModelArg(t *testing.T) {
	// --model is not reserved, so a per-reviewer model override is accepted.
	path := writeGlobalConfig(t, `
review:
  reviewers:
    - agent: claude
      args: ["--model", "claude-opus-4-8"]
    - agent: claude
      args: ["--model", "claude-haiku-4-5-20251001"]
      label: claude-haiku
`)
	cfg, err := LoadGlobal(path)
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if len(cfg.Reviewers) != 2 {
		t.Fatalf("want 2 reviewers, got %d", len(cfg.Reviewers))
	}
	if len(cfg.Reviewers[0].Args) != 2 || cfg.Reviewers[0].Args[1] != "claude-opus-4-8" {
		t.Errorf("reviewer 0 args = %+v", cfg.Reviewers[0].Args)
	}
}
