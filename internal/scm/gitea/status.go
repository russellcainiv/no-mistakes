package gitea

import (
	"context"
	"fmt"
	"strings"
)

// GateContext is the commit-status context that Forgejo/Gitea branch
// protection matches on as a required status check.
const GateContext = "no-mistakes/gate"

// PostGateStatus stamps a gate outcome as a GateContext commit status on the
// Gitea/Forgejo repo identified by remoteURL. It returns (false, nil) when
// remoteURL or sha is empty — nothing to stamp — so callers can treat the
// stamp as best effort without special-casing unconfigured mirrors.
func PostGateStatus(ctx context.Context, remoteURL, sha, prURL string, success bool) (bool, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	sha = strings.TrimSpace(sha)
	if remoteURL == "" || sha == "" {
		return false, nil
	}
	ref, err := ParseRepoRef(remoteURL)
	if err != nil {
		return false, fmt.Errorf("parse forgejo remote %q: %w", remoteURL, err)
	}
	client, err := NewClientFromEnv(nil, ref)
	if err != nil {
		return false, err
	}
	state, desc := "success", "no-mistakes gate passed"
	if !success {
		state, desc = "failure", "no-mistakes gate did not pass"
	}
	if err := client.SetCommitStatus(ctx, ref, sha, state, GateContext, desc, prURL); err != nil {
		return false, err
	}
	return true, nil
}
