// Package gitea implements the scm.Host interface for Gitea and Forgejo
// instances via their shared /api/v1 REST API. Forgejo is a Gitea fork and
// its API is wire-compatible, so a single client serves both.
package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// envToken overrides the API token used for authentication. When unset the
	// client falls back to the token recorded by the `tea` CLI for the matching
	// host in its config.yml.
	envToken = "NO_MISTAKES_GITEA_API_TOKEN"
	// envBaseURL overrides the API base URL (scheme://host[:port]). When unset
	// the base is derived from the repository remote URL. Required for SSH
	// remotes, whose scheme/port cannot be inferred.
	envBaseURL = "NO_MISTAKES_GITEA_API_BASE_URL"
)

// RepoRef identifies a repository on a Gitea/Forgejo instance.
type RepoRef struct {
	BaseURL string // scheme://host[:port], no trailing slash
	Owner   string
	Repo    string
}

// PullRequest is the subset of the Gitea pull object this tool consumes.
type PullRequest struct {
	Number    int    `json:"number"`
	HTMLURL   string `json:"html_url"`
	State     string `json:"state"`
	Merged    bool   `json:"merged"`
	Mergeable bool   `json:"mergeable"`
	Head      struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// CommitStatus is a single CI status attached to a commit. Forgejo Actions and
// Gitea Actions both report workflow/job results as commit statuses.
type CommitStatus struct {
	Context   string `json:"context"`
	State     string `json:"status"`
	TargetURL string `json:"target_url"`
	UpdatedAt string `json:"updated_at"`
}

// Client is a Gitea/Forgejo REST API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClientFromEnv builds a client for repo, resolving the token from the
// environment (NO_MISTAKES_GITEA_API_TOKEN) first and the `tea` CLI config
// second. The base URL comes from repo.BaseURL, overridable via
// NO_MISTAKES_GITEA_API_BASE_URL. It returns an error when no token or base URL
// can be resolved, so callers fail closed rather than issuing anonymous calls.
func NewClientFromEnv(env []string, repo RepoRef) (*Client, error) {
	baseURL := lookupEnv(env, envBaseURL)
	if strings.TrimSpace(baseURL) == "" {
		baseURL = repo.BaseURL
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("no Gitea/Forgejo API base URL; set %s", envBaseURL)
	}

	token := lookupEnv(env, envToken)
	if strings.TrimSpace(token) == "" {
		token = tokenFromTeaConfig(hostOf(baseURL))
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("no Gitea/Forgejo token; set %s or run `tea login add`", envToken)
	}

	return &Client{
		baseURL:    baseURL,
		token:      strings.TrimSpace(token),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// FindOpenPRByHead returns the open PR whose head ref matches branch (and, when
// base is non-empty, whose base ref matches base), or nil when none exists.
// Gitea's list-pulls endpoint has no head filter, so results are filtered
// client-side.
func (c *Client) FindOpenPRByHead(ctx context.Context, repo RepoRef, branch, base string) (*PullRequest, error) {
	var pulls []PullRequest
	path := fmt.Sprintf("%s?state=open&limit=50", repoPullsPath(repo))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &pulls); err != nil {
		return nil, err
	}
	for i := range pulls {
		if pulls[i].Head.Ref != branch {
			continue
		}
		if strings.TrimSpace(base) != "" && pulls[i].Base.Ref != base {
			continue
		}
		return &pulls[i], nil
	}
	return nil, nil
}

// CreatePR opens a pull request. head is the source branch; when forkOwner is
// non-empty the head is qualified as "forkOwner:branch" for a cross-repo PR.
func (c *Client) CreatePR(ctx context.Context, repo RepoRef, branch, base, forkOwner, title, body string) (*PullRequest, error) {
	head := branch
	if strings.TrimSpace(forkOwner) != "" {
		head = forkOwner + ":" + branch
	}
	reqBody := map[string]any{"head": head, "base": base, "title": title, "body": body}
	var pr PullRequest
	if err := c.doJSON(ctx, http.MethodPost, repoPullsPath(repo), reqBody, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// UpdatePR edits an existing pull request's title and body.
func (c *Client) UpdatePR(ctx context.Context, repo RepoRef, number int, title, body string) (*PullRequest, error) {
	reqBody := map[string]any{"title": title, "body": body}
	var pr PullRequest
	if err := c.doJSON(ctx, http.MethodPatch, fmt.Sprintf("%s/%d", repoPullsPath(repo), number), reqBody, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetPR fetches a single pull request by number.
func (c *Client) GetPR(ctx context.Context, repo RepoRef, number int) (*PullRequest, error) {
	var pr PullRequest
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/%d", repoPullsPath(repo), number), nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// ListCommitStatuses returns the CI statuses attached to a commit SHA, newest
// first.
func (c *Client) ListCommitStatuses(ctx context.Context, repo RepoRef, sha string) ([]CommitStatus, error) {
	var statuses []CommitStatus
	path := fmt.Sprintf("/api/v1/repos/%s/%s/commits/%s/statuses?limit=50&sort=recentupdate",
		url.PathEscape(repo.Owner), url.PathEscape(repo.Repo), url.PathEscape(sha))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}

// SetCommitStatus posts a commit status to a SHA. state is one of Gitea's
// status states: pending, success, error, failure, warning. statusContext is
// the check name (e.g. "no-mistakes/gate") that branch protection matches on.
func (c *Client) SetCommitStatus(ctx context.Context, repo RepoRef, sha, state, statusContext, description, targetURL string) error {
	body := map[string]any{
		"state":       state,
		"context":     statusContext,
		"description": description,
	}
	if strings.TrimSpace(targetURL) != "" {
		body["target_url"] = targetURL
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/statuses/%s",
		url.PathEscape(repo.Owner), url.PathEscape(repo.Repo), url.PathEscape(sha))
	return c.doJSON(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestBody, responseBody any) error {
	var bodyReader io.Reader = http.NoBody
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal Gitea request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build Gitea request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+c.token)
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Gitea %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Gitea %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode Gitea response: %w", err)
	}
	return nil
}

func repoPullsPath(repo RepoRef) string {
	return fmt.Sprintf("/api/v1/repos/%s/%s/pulls", url.PathEscape(repo.Owner), url.PathEscape(repo.Repo))
}

// ParseRepoRef extracts the base URL, owner, and repo from a Gitea/Forgejo
// remote URL. It handles https URLs (scheme + host + port preserved) and
// scp-like SSH syntax (git@host:owner/repo.git), for which the API base falls
// back to https://host and must usually be overridden via env for a custom
// port or an http instance.
func ParseRepoRef(raw string) (RepoRef, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimSuffix(trimmed, ".git")
	if trimmed == "" {
		return RepoRef{}, fmt.Errorf("empty Gitea remote URL")
	}

	// scp-like: [user@]host:owner/repo
	if !strings.Contains(trimmed, "://") {
		at := strings.LastIndex(trimmed, "@")
		hostPath := trimmed
		if at >= 0 {
			hostPath = trimmed[at+1:]
		}
		colon := strings.Index(hostPath, ":")
		if colon < 0 {
			return RepoRef{}, fmt.Errorf("invalid Gitea SSH remote %q", raw)
		}
		host := hostPath[:colon]
		owner, repo, err := splitOwnerRepo(hostPath[colon+1:])
		if err != nil {
			return RepoRef{}, err
		}
		return RepoRef{BaseURL: "https://" + host, Owner: owner, Repo: repo}, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return RepoRef{}, fmt.Errorf("parse Gitea remote URL: %w", err)
	}
	owner, repo, err := splitOwnerRepo(strings.TrimPrefix(parsed.Path, "/"))
	if err != nil {
		return RepoRef{}, err
	}
	return RepoRef{
		BaseURL: parsed.Scheme + "://" + parsed.Host,
		Owner:   owner,
		Repo:    repo,
	}, nil
}

func splitOwnerRepo(path string) (string, string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid Gitea repository path %q", path)
	}
	return parts[0], parts[1], nil
}

func hostOf(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Host)
}

// tokenFromTeaConfig returns the token the `tea` CLI recorded for the login
// whose URL host matches host, or "" when tea has no matching login. Any
// read/parse error yields "" so token resolution fails closed.
func tokenFromTeaConfig(host string) string {
	if host == "" {
		return ""
	}
	path := teaConfigPath()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		Logins []struct {
			URL   string `yaml:"url"`
			Token string `yaml:"token"`
		} `yaml:"logins"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	host = strings.ToLower(host)
	for _, login := range cfg.Logins {
		if hostOf(strings.TrimSpace(login.URL)) == host {
			return strings.TrimSpace(login.Token)
		}
	}
	return ""
}

// teaConfigPath resolves tea's config.yml. It returns the first candidate that
// exists, checking $TEA_CONFIG_HOME, $XDG_CONFIG_HOME/tea, the macOS
// Application Support location (where tea writes when XDG_CONFIG_HOME is unset),
// and finally ~/.config/tea. When none exist it returns the last candidate so
// callers have a stable path to report.
func teaConfigPath() string {
	var candidates []string
	if dir := os.Getenv("TEA_CONFIG_HOME"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, "config.yml"))
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, "tea", "config.yml"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, "Library", "Application Support", "tea", "config.yml"),
			filepath.Join(home, ".config", "tea", "config.yml"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if len(candidates) > 0 {
		return candidates[len(candidates)-1]
	}
	return ""
}

func lookupEnv(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return ""
}
