package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/scm"
)

func TestParseRepoRef(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		wantBase  string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{name: "http with port", raw: "http://localhost:3000/russell/lvl2.git", wantBase: "http://localhost:3000", wantOwner: "russell", wantRepo: "lvl2"},
		{name: "https no port", raw: "https://codeberg.org/owner/proj", wantBase: "https://codeberg.org", wantOwner: "owner", wantRepo: "proj"},
		{name: "scp ssh", raw: "git@gitea.example.com:team/service.git", wantBase: "https://gitea.example.com", wantOwner: "team", wantRepo: "service"},
		{name: "missing repo", raw: "http://localhost:3000/russell", wantErr: true},
		{name: "empty", raw: "  ", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRepoRef(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %+v", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.BaseURL != tc.wantBase || got.Owner != tc.wantOwner || got.Repo != tc.wantRepo {
				t.Fatalf("got %+v, want base=%s owner=%s repo=%s", got, tc.wantBase, tc.wantOwner, tc.wantRepo)
			}
		})
	}
}

func TestStatusBucket(t *testing.T) {
	cases := map[string]scm.CheckBucket{
		"success": scm.CheckBucketPass,
		"warning": scm.CheckBucketPass,
		"failure": scm.CheckBucketFail,
		"error":   scm.CheckBucketFail,
		"pending": scm.CheckBucketPending,
		"weird":   "",
	}
	for state, want := range cases {
		if got := statusBucket(state); got != want {
			t.Errorf("statusBucket(%q) = %q, want %q", state, got, want)
		}
	}
}

func TestToChecksKeepsNewestPerContext(t *testing.T) {
	// ListCommitStatuses returns newest-first; the first status per context wins.
	statuses := []CommitStatus{
		{Context: "build", State: "success"},
		{Context: "build", State: "failure"}, // stale, ignored
		{Context: "lint", State: "pending"},
	}
	checks := toChecks(statuses)
	if len(checks) != 2 {
		t.Fatalf("want 2 checks, got %d: %+v", len(checks), checks)
	}
	if checks[0].Name != "build" || checks[0].Bucket != scm.CheckBucketPass {
		t.Errorf("build check = %+v, want pass", checks[0])
	}
	if checks[1].Name != "lint" || checks[1].Bucket != scm.CheckBucketPending {
		t.Errorf("lint check = %+v, want pending", checks[1])
	}
}

func TestToChecksExcludesOwnGateStatus(t *testing.T) {
	// The gate's own stamp must not feed back into the CI monitor.
	statuses := []CommitStatus{
		{Context: GateContext, State: "failure"},
		{Context: "build", State: "success"},
	}
	checks := toChecks(statuses)
	if len(checks) != 1 || checks[0].Name != "build" {
		t.Fatalf("want only the build check, got %+v", checks)
	}
}

func TestToChecksMapsUpdatedAtToCompletedAt(t *testing.T) {
	// Finished statuses carry UpdatedAt as their completion time (the CI
	// monitor's re-run detection keys on it); pending ones stay zero.
	statuses := []CommitStatus{
		{Context: "build", State: "failure", UpdatedAt: "2026-07-09T22:00:00Z"},
		{Context: "lint", State: "pending", UpdatedAt: "2026-07-09T22:01:00Z"},
	}
	checks := toChecks(statuses)
	if len(checks) != 2 {
		t.Fatalf("want 2 checks, got %+v", checks)
	}
	if checks[0].CompletedAt.IsZero() {
		t.Errorf("failing check CompletedAt is zero, want parsed UpdatedAt")
	}
	if !checks[1].CompletedAt.IsZero() {
		t.Errorf("pending check CompletedAt = %v, want zero", checks[1].CompletedAt)
	}
}

func TestNormalizePRState(t *testing.T) {
	merged := &PullRequest{State: "closed", Merged: true}
	if got := normalizePRState(merged); got != scm.PRStateMerged {
		t.Errorf("merged PR = %q, want MERGED", got)
	}
	closed := &PullRequest{State: "closed"}
	if got := normalizePRState(closed); got != scm.PRStateClosed {
		t.Errorf("closed PR = %q, want CLOSED", got)
	}
	open := &PullRequest{State: "open"}
	if got := normalizePRState(open); got != scm.PRStateOpen {
		t.Errorf("open PR = %q, want OPEN", got)
	}
}

// newTestHost spins up an httptest server that emulates the Gitea endpoints
// this tool uses, and returns a Host wired to it.
func newTestHost(t *testing.T, handler http.HandlerFunc) (*Host, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client := &Client{baseURL: srv.URL, token: "test-token", httpClient: srv.Client()}
	return NewHost(client, RepoRef{BaseURL: srv.URL, Owner: "russell", Repo: "lvl2"}, ""), srv
}

func TestHostCreateAndFindPR(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod, gotPath = r.Method, r.URL.Path
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["head"] != "feat/x" || body["base"] != "main" {
				t.Errorf("unexpected create body: %+v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PullRequest{Number: 7, HTMLURL: "http://localhost:3000/russell/lvl2/pulls/7", State: "open"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pulls"):
			pulls := []PullRequest{{Number: 7, State: "open"}}
			pulls[0].Head.Ref = "feat/x"
			pulls[0].Base.Ref = "main"
			_ = json.NewEncoder(w).Encode(pulls)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}
	host, srv := newTestHost(t, handler)
	defer srv.Close()
	ctx := context.Background()

	created, err := host.CreatePR(ctx, "feat/x", "main", scm.PRContent{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}
	if created.Number != "7" || created.URL == "" {
		t.Errorf("created PR = %+v", created)
	}
	if gotAuth != "token test-token" {
		t.Errorf("auth header = %q, want token test-token", gotAuth)
	}

	found, err := host.FindPR(ctx, "feat/x", "main")
	if err != nil {
		t.Fatalf("FindPR: %v", err)
	}
	if found == nil || found.Number != "7" {
		t.Errorf("FindPR = %+v", found)
	}
	// A branch with no matching head returns nil, not an error.
	none, err := host.FindPR(ctx, "other", "main")
	if err != nil || none != nil {
		t.Errorf("FindPR(no match) = %+v, %v; want nil, nil", none, err)
	}
	_ = gotMethod
	_ = gotPath
}

func TestHostGetChecks(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pulls/7"):
			pr := PullRequest{Number: 7, State: "open"}
			pr.Head.SHA = "abc123"
			_ = json.NewEncoder(w).Encode(pr)
		case strings.Contains(r.URL.Path, "/commits/abc123/statuses"):
			_ = json.NewEncoder(w).Encode([]CommitStatus{
				{Context: "web-ci", State: "success"},
				{Context: "react-doctor", State: "failure"},
			})
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}
	host, srv := newTestHost(t, handler)
	defer srv.Close()

	checks, err := host.GetChecks(context.Background(), &scm.PR{Number: "7"})
	if err != nil {
		t.Fatalf("GetChecks: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("want 2 checks, got %+v", checks)
	}
	byName := map[string]scm.CheckBucket{}
	for _, c := range checks {
		byName[c.Name] = c.Bucket
	}
	if byName["web-ci"] != scm.CheckBucketPass || byName["react-doctor"] != scm.CheckBucketFail {
		t.Errorf("unexpected check buckets: %+v", byName)
	}
}
