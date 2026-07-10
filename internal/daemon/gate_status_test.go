package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
)

// TestPostGateStatusTerminalPost exercises the daemon's terminal stamp: when
// the registered upstream is a Gitea/Forgejo URL, the outcome lands there as
// a no-mistakes/gate commit status on the run's head SHA.
func TestPostGateStatusTerminalPost(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	t.Setenv("NO_MISTAKES_GITEA_API_TOKEN", "tok")
	// Allowlist the httptest host so provider detection resolves the bare
	// 127.0.0.1 upstream to Gitea, matching a localhost Forgejo setup.
	t.Setenv("NO_MISTAKES_GITEA_HOSTS", "127.0.0.1")

	prURL := "http://pr/1"
	run := &db.Run{ID: "run-1", HeadSHA: "abc123", PRURL: &prURL}
	repo := &db.Repo{UpstreamURL: srv.URL + "/russell/lvl2.git"}
	postGateStatus(run, repo, t.TempDir(), false)

	if gotPath != "/api/v1/repos/russell/lvl2/statuses/abc123" {
		t.Errorf("path = %q, want statuses post for head SHA", gotPath)
	}
	if gotBody["state"] != "failure" || gotBody["context"] != "no-mistakes/gate" || gotBody["target_url"] != prURL {
		t.Errorf("body = %+v", gotBody)
	}
}
