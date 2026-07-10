package scm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProviderGiteaMarkers(t *testing.T) {
	cases := map[string]Provider{
		"https://gitea.example.com/o/r.git": ProviderGitea,
		"https://forgejo.example.org/o/r":   ProviderGitea,
		"https://codeberg.org/o/r.git":      ProviderGitea,
		"git@gitea.acme.io:team/svc.git":    ProviderGitea,
		"https://github.com/o/r.git":        ProviderGitHub,
	}
	for url, want := range cases {
		if got := DetectProvider(url); got != want {
			t.Errorf("DetectProvider(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestDetectProviderGiteaEnvHosts(t *testing.T) {
	// Hermetic: ignore this machine's real tea logins and env allowlist, both
	// of which can legitimately mark localhost as Gitea outside the test.
	// teaConfigPath falls through to the first config.yml that exists, so the
	// stub file must exist or the real user config would still be read.
	teaDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(teaDir, "config.yml"), []byte("logins: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEA_CONFIG_HOME", teaDir)
	t.Setenv("NO_MISTAKES_GITEA_HOSTS", "")
	// A bare host with no marker resolves to Gitea only when listed in
	// NO_MISTAKES_GITEA_HOSTS. Port is ignored in the comparison.
	if got := DetectProvider("http://localhost:3000/russell/lvl2.git"); got == ProviderGitea {
		t.Fatalf("localhost should not be Gitea without env allowlist, got %q", got)
	}
	t.Setenv("NO_MISTAKES_GITEA_HOSTS", "localhost")
	if got := DetectProvider("http://localhost:3000/russell/lvl2.git"); got != ProviderGitea {
		t.Errorf("with allowlist, localhost:3000 = %q, want gitea", got)
	}
}
