package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestSetCommitStatus checks the request shape against a stub server.
func TestSetCommitStatus(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	ref := RepoRef{BaseURL: srv.URL, Owner: "russell", Repo: "lvl2"}
	err := client.SetCommitStatus(context.Background(), ref, "abc123", "success", "no-mistakes/gate", "gate passed", "http://pr")
	if err != nil {
		t.Fatalf("SetCommitStatus: %v", err)
	}
	if gotPath != "/api/v1/repos/russell/lvl2/statuses/abc123" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "token tok" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotBody["state"] != "success" || gotBody["context"] != "no-mistakes/gate" || gotBody["target_url"] != "http://pr" {
		t.Errorf("body = %+v", gotBody)
	}
}

// TestLiveSetCommitStatus exercises the exact daemon path — build a client via
// NewClientFromEnv (which resolves the token from env or the tea config) and
// POST a real status to a real Forgejo commit. Gated on NM_LIVE_FORGEJO_SHA so
// it only runs when explicitly asked.
func TestLiveSetCommitStatus(t *testing.T) {
	sha := os.Getenv("NM_LIVE_FORGEJO_SHA")
	if strings.TrimSpace(sha) == "" {
		t.Skip("set NM_LIVE_FORGEJO_SHA to run the live Forgejo smoke test")
	}
	ref, err := ParseRepoRef("http://localhost:3000/russell/lvl2.git")
	if err != nil {
		t.Fatalf("ParseRepoRef: %v", err)
	}
	client, err := NewClientFromEnv(nil, ref)
	if err != nil {
		t.Fatalf("NewClientFromEnv (token resolution): %v", err)
	}
	if err := client.SetCommitStatus(context.Background(), ref, sha, "success", "no-mistakes/gate", "live smoke test", ""); err != nil {
		t.Fatalf("SetCommitStatus (live): %v", err)
	}
}
