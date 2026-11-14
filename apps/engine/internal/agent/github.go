package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHub is a thin REST client for the bits of GitHub the agent needs: read
// files and commits, then push a fix branch and open a PR.
type GitHub struct {
	Token   string
	BaseURL string
	HTTP    *http.Client
}

// Commit is one recent commit on the repo default branch.
type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

func (g *GitHub) base() string {
	if g.BaseURL == "" {
		return "https://api.github.com"
	}
	return strings.TrimSuffix(g.BaseURL, "/")
}

func (g *GitHub) client() *http.Client {
	if g.HTTP == nil {
		g.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	return g.HTTP
}

func (g *GitHub) do(ctx context.Context, method, path string, body any, accept string) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.base()+path, reader)
	if err != nil {
		return nil, 0, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return raw, resp.StatusCode, fmt.Errorf("github %s %s: %s", method, path, resp.Status)
	}
	return raw, resp.StatusCode, nil
}

// DefaultBranch returns the repo default branch name.
func (g *GitHub) DefaultBranch(ctx context.Context, repo string) (string, error) {
	raw, _, err := g.do(ctx, http.MethodGet, "/repos/"+repo, nil, "")
	if err != nil {
		return "", err
	}
	var out struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.DefaultBranch == "" {
		return "main", nil
	}
	return out.DefaultBranch, nil
}

// FileContent reads a file from the repo default branch as plain text.
func (g *GitHub) FileContent(ctx context.Context, repo, path string) (string, error) {
	p := "/repos/" + repo + "/contents/" + strings.TrimPrefix(path, "/")
	raw, _, err := g.do(ctx, http.MethodGet, p, nil, "application/vnd.github.raw+json")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// RecentCommits lists the newest commit messages on the default branch.
func (g *GitHub) RecentCommits(ctx context.Context, repo string, n int) ([]Commit, error) {
	if n <= 0 {
		n = 5
	}
	raw, _, err := g.do(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/commits?per_page=%d", repo, n), nil, "")
	if err != nil {
		return nil, err
	}
	var items []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	out := make([]Commit, 0, len(items))
	for _, it := range items {
		msg := it.Commit.Message
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		out = append(out, Commit{SHA: it.SHA, Message: msg})
	}
	return out, nil
}

// CreateBranch cuts a new branch from the default branch head.
func (g *GitHub) CreateBranch(ctx context.Context, repo, branch string) (string, error) {
	base, err := g.DefaultBranch(ctx, repo)
	if err != nil {
		return "", err
	}
	raw, _, err := g.do(ctx, http.MethodGet, "/repos/"+repo+"/git/ref/heads/"+base, nil, "")
	if err != nil {
		return "", err
	}
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		return "", err
	}
	_, _, err = g.do(ctx, http.MethodPost, "/repos/"+repo+"/git/refs", map[string]string{
		"ref": "refs/heads/" + branch,
		"sha": ref.Object.SHA,
	}, "")
	if err != nil {
		return "", err
	}
	return ref.Object.SHA, nil
}

// CommitFile writes (or replaces) one file on a branch.
func (g *GitHub) CommitFile(ctx context.Context, repo, branch, path, content, message string) error {
	p := "/repos/" + repo + "/contents/" + strings.TrimPrefix(path, "/")
	sha := ""
	if raw, _, err := g.do(ctx, http.MethodGet, p+"?ref="+url.QueryEscape(branch), nil, ""); err == nil {
		var meta struct {
			SHA string `json:"sha"`
		}
		_ = json.Unmarshal(raw, &meta)
		sha = meta.SHA
	}
	body := map[string]string{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  branch,
	}
	if sha != "" {
		body["sha"] = sha
	}
	_, _, err := g.do(ctx, http.MethodPut, p, body, "")
	return err
}

// OpenPR opens a pull request and returns its web URL.
func (g *GitHub) OpenPR(ctx context.Context, repo, title, head, body string) (string, error) {
	base, err := g.DefaultBranch(ctx, repo)
	if err != nil {
		return "", err
	}
	raw, _, err := g.do(ctx, http.MethodPost, "/repos/"+repo+"/pulls", map[string]string{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
	}, "")
	if err != nil {
		return "", err
	}
	var pr struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(raw, &pr); err != nil {
		return "", err
	}
	return pr.HTMLURL, nil
}
