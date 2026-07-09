package scm

import "testing"

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
