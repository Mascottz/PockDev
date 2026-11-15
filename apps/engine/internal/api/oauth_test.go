package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/auth"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

// fakeGitHub serves the three GitHub endpoints the OAuth flow touches.
func fakeGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.Form.Get("code") == "" || r.Form.Get("client_id") == "" || r.Form.Get("client_secret") == "" {
			t.Errorf("missing oauth form fields: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_test", "token_type": "bearer"})
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer gho_test" {
			t.Errorf("bad authorization header %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 1001, "login": "Mascottz", "name": "Mascott", "email": "",
			"avatar_url": "https://avatars.test/1.png",
		})
	})
	mux.HandleFunc("GET /user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "noise@x.com", "primary": false, "verified": false},
			{"email": "mascottz@example.com", "primary": true, "verified": true},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func oauthSetup(t *testing.T) (*API, *memstore.Mem, *auth.Handler, http.Handler) {
	t.Helper()
	st := memstore.New()
	gh := fakeGitHub(t)
	h := &auth.Handler{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		OAuthBaseURL: gh.URL,
		APIBaseURL:   gh.URL,
		AdminLogin:   "Mascottz",
		Store:        st,
	}
	a := &API{Store: st, APIToken: testToken, StartedAt: time.Now()}
	mux := http.NewServeMux()
	a.Register(mux)
	h.Register(mux)
	return a, st, h, mux
}

func TestGitHubOAuthFlow(t *testing.T) {
	_, st, _, h := oauthSetup(t)

	// 1: start issues a signed state and redirects to GitHub authorize.
	rec := do(t, h, http.MethodGet, "/auth/github/start", "", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("start: %d %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(loc.String(), "http://127.0.0.1:") || loc.Path != "/login/oauth/authorize" {
		t.Fatalf("authorize redirect %q", loc.String())
	}
	q := loc.Query()
	if q.Get("client_id") != "client-id" || q.Get("scope") != "read:user,user:email" {
		t.Fatalf("authorize query %v", q)
	}
	state := q.Get("state")
	if state == "" {
		t.Fatal("state missing")
	}

	// 2: callback exchanges the code and redirects with the session fragment.
	rec = do(t, h, http.MethodGet, "/auth/github/callback?code=abc123&state="+url.QueryEscape(state), "", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("callback: %d %s", rec.Code, rec.Body.String())
	}
	cbLoc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cbLoc.String(), "http://localhost:3000/auth/callback#session=pds_") {
		t.Fatalf("session redirect %q", cbLoc.String())
	}
	rawToken := strings.TrimPrefix(cbLoc.Fragment, "session=")
	if !strings.HasPrefix(rawToken, "pds_") {
		t.Fatalf("raw token %q", rawToken)
	}

	// The stored session only holds the hash and points at the claimed
	// default workspace.
	sess, user, err := st.GetSession(context.Background(), auth.HashToken(rawToken), time.Now())
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if user.GitHubLogin != "Mascottz" || user.Email != "mascottz@example.com" {
		t.Fatalf("user %+v", user)
	}
	if sess.WorkspaceID != store.DefaultWorkspaceID {
		t.Fatalf("workspace %q", sess.WorkspaceID)
	}

	// 3: /api/me reports the session, user, workspace, and memberships.
	rec = do(t, h, http.MethodGet, "/api/me", rawToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("me: %d %s", rec.Code, rec.Body.String())
	}
	var me map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me["mode"] != "session" {
		t.Fatalf("mode %v", me["mode"])
	}
	u := me["user"].(map[string]any)
	if u["githubLogin"] != "Mascottz" || u["email"] != "mascottz@example.com" || u["id"] == "" {
		t.Fatalf("user %v", u)
	}
	wsp := me["workspace"].(map[string]any)
	if wsp["id"] != store.DefaultWorkspaceID || wsp["name"] != "Mascottz Workspace" || wsp["slug"] != "default" {
		t.Fatalf("workspace %v", wsp)
	}
	list := me["workspaces"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["role"] != "owner" {
		t.Fatalf("workspaces %v", list)
	}

	// 4: creating a workspace works in session mode.
	rec = do(t, h, http.MethodPost, "/api/workspaces", rawToken, `{"name": "Side Project"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create workspace: %d %s", rec.Code, rec.Body.String())
	}
	var wsp2 map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &wsp2); err != nil {
		t.Fatal(err)
	}
	if wsp2["slug"] != "side-project" || wsp2["role"] != "owner" {
		t.Fatalf("workspace %v", wsp2)
	}

	// 5: logout kills the session.
	rec = do(t, h, http.MethodPost, "/api/auth/logout", rawToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body.String())
	}
	rec = do(t, h, http.MethodGet, "/api/me", rawToken, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout: %d %s", rec.Code, rec.Body.String())
	}
	if _, _, err := st.GetSession(context.Background(), auth.HashToken(rawToken), time.Now()); err == nil {
		t.Fatal("session must be deleted")
	}
}

func TestGitHubOAuthRejectsBadStateAndRedirect(t *testing.T) {
	_, _, _, h := oauthSetup(t)

	rec := do(t, h, http.MethodGet, "/auth/github/start?redirect_to=https://evil.com/steal", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("untrusted redirect: %d", rec.Code)
	}

	rec = do(t, h, http.MethodGet, "/auth/github/callback?code=abc&state=garbage", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad state: %d", rec.Code)
	}

	rec = do(t, h, http.MethodGet, "/auth/github/callback?state=garbage", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing code: %d", rec.Code)
	}
}

func TestLogoutWithoutSessionStillOK(t *testing.T) {
	_, _, _, h := oauthSetup(t)
	rec := do(t, h, http.MethodPost, "/api/auth/logout", "random-token", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMeForTokenMode(t *testing.T) {
	a, _, _, h := oauthSetup(t)
	_ = a
	rec := do(t, h, http.MethodGet, "/api/me", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("me: %d %s", rec.Code, rec.Body.String())
	}
	var me map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me["mode"] != "token" || me["user"] != nil {
		t.Fatalf("token me %v", me)
	}
	if me["workspace"].(map[string]any)["id"] != store.DefaultWorkspaceID {
		t.Fatalf("workspace %v", me["workspace"])
	}
	if list := me["workspaces"].([]any); len(list) != 0 {
		t.Fatalf("token workspaces %v", list)
	}

	// Workspace creation requires a real session.
	rec = do(t, h, http.MethodPost, "/api/workspaces", testToken, `{"name": "Nope"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("token create workspace: %d", rec.Code)
	}
}
