package gitea

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/scm"
)

// Host implements scm.Host for Gitea/Forgejo via the REST API client.
type Host struct {
	client    *Client
	repo      RepoRef
	forkOwner string
}

// NewHost builds a Host from an API client, a repository reference, and an
// optional fork owner (non-empty when PRs are opened from a fork's branch).
func NewHost(client *Client, repo RepoRef, forkOwner string) *Host {
	return &Host{client: client, repo: repo, forkOwner: forkOwner}
}

func (h *Host) Provider() scm.Provider { return scm.ProviderGitea }

// Capabilities reports Gitea/Forgejo's feature matrix. The pull object exposes
// a mergeable flag, so conflict detection is supported. Per-check log retrieval
// through the Actions API is not implemented yet.
func (h *Host) Capabilities() scm.Capabilities {
	return scm.Capabilities{MergeableState: true, FailedCheckLogs: false}
}

func (h *Host) Available(_ context.Context) error {
	if h.client == nil {
		return errors.New("gitea client is not configured")
	}
	return nil
}

func (h *Host) FindPR(ctx context.Context, branch, base string) (*scm.PR, error) {
	pr, err := h.client.FindOpenPRByHead(ctx, h.repo, branch, base)
	if err != nil {
		return nil, err
	}
	return h.toPR(pr), nil
}

func (h *Host) CreatePR(ctx context.Context, branch, base string, content scm.PRContent) (*scm.PR, error) {
	pr, err := h.client.CreatePR(ctx, h.repo, branch, base, h.forkOwner, content.Title, content.Body)
	if err != nil {
		return nil, err
	}
	return h.toPR(pr), nil
}

func (h *Host) UpdatePR(ctx context.Context, pr *scm.PR, content scm.PRContent) (*scm.PR, error) {
	number, err := strconv.Atoi(pr.Number)
	if err != nil {
		return nil, fmt.Errorf("invalid Gitea PR number %q: %w", pr.Number, err)
	}
	updated, err := h.client.UpdatePR(ctx, h.repo, number, content.Title, content.Body)
	if err != nil {
		return nil, err
	}
	return h.toPR(updated), nil
}

func (h *Host) GetPRState(ctx context.Context, pr *scm.PR) (scm.PRState, error) {
	number, err := strconv.Atoi(pr.Number)
	if err != nil {
		return "", err
	}
	got, err := h.client.GetPR(ctx, h.repo, number)
	if err != nil {
		return "", err
	}
	if got == nil {
		return "", nil
	}
	return normalizePRState(got), nil
}

func (h *Host) GetChecks(ctx context.Context, pr *scm.PR) ([]scm.Check, error) {
	number, err := strconv.Atoi(pr.Number)
	if err != nil {
		return nil, err
	}
	got, err := h.client.GetPR(ctx, h.repo, number)
	if err != nil {
		return nil, err
	}
	if got == nil || strings.TrimSpace(got.Head.SHA) == "" {
		return nil, nil
	}
	statuses, err := h.client.ListCommitStatuses(ctx, h.repo, got.Head.SHA)
	if err != nil {
		return nil, err
	}
	return toChecks(statuses), nil
}

func (h *Host) GetMergeableState(ctx context.Context, pr *scm.PR) (scm.MergeableState, error) {
	number, err := strconv.Atoi(pr.Number)
	if err != nil {
		return "", err
	}
	got, err := h.client.GetPR(ctx, h.repo, number)
	if err != nil {
		return "", err
	}
	if got == nil {
		return scm.MergeableUnknown, nil
	}
	if got.Mergeable {
		return scm.MergeableOK, nil
	}
	return scm.MergeableConflict, nil
}

func (h *Host) FetchFailedCheckLogs(_ context.Context, _ *scm.PR, _ string, _ string, _ []string) (string, error) {
	return "", scm.ErrUnsupported
}

func (h *Host) toPR(pr *PullRequest) *scm.PR {
	if pr == nil {
		return nil
	}
	return &scm.PR{Number: strconv.Itoa(pr.Number), URL: strings.TrimSpace(pr.HTMLURL)}
}

func normalizePRState(pr *PullRequest) scm.PRState {
	if pr.Merged {
		return scm.PRStateMerged
	}
	switch strings.ToLower(strings.TrimSpace(pr.State)) {
	case "open":
		return scm.PRStateOpen
	case "closed":
		return scm.PRStateClosed
	default:
		return scm.PRState(pr.State)
	}
}

// toChecks collapses commit statuses to the newest status per context and maps
// each to a normalized check. ListCommitStatuses returns newest-first, so the
// first status seen for a context is the current one. no-mistakes' own gate
// stamp is excluded: it is not CI, and feeding it back into the CI monitor
// would make no-mistakes react to its own outcome (e.g. auto-fixing a stale
// gate failure left by a previous run on the same commit).
func toChecks(statuses []CommitStatus) []scm.Check {
	seen := make(map[string]struct{}, len(statuses))
	checks := make([]scm.Check, 0, len(statuses))
	for _, status := range statuses {
		name := strings.TrimSpace(status.Context)
		if name == GateContext {
			continue
		}
		if name != "" {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
		}
		bucket := statusBucket(status.State)
		// UpdatedAt doubles as the completion time for finished statuses; the
		// CI monitor uses it to detect a re-run that failed again between
		// polls (failingCheckCompletedAfter). Leave it zero while pending.
		completedAt := time.Time{}
		if bucket == scm.CheckBucketPass || bucket == scm.CheckBucketFail {
			if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(status.UpdatedAt)); err == nil {
				completedAt = parsed
			}
		}
		checks = append(checks, scm.Check{Name: name, Bucket: bucket, CompletedAt: completedAt})
	}
	return checks
}

func statusBucket(state string) scm.CheckBucket {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "success":
		return scm.CheckBucketPass
	case "warning":
		// Gitea "warning" is a non-blocking soft-pass state.
		return scm.CheckBucketPass
	case "failure", "error":
		return scm.CheckBucketFail
	case "pending":
		return scm.CheckBucketPending
	default:
		return ""
	}
}
