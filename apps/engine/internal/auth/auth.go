// Package auth implements the GitHub OAuth sign in flow and session tokens
// for the PockDev engine.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// SessionTokenPrefix marks raw session tokens issued by the callback flow.
const SessionTokenPrefix = "pds_"

// SessionTTL is how long an issued session token stays valid.
const SessionTTL = 30 * 24 * time.Hour

// stateTTL is how long a signed OAuth state stays fresh.
const stateTTL = 10 * time.Minute

// HashToken returns the SHA-256 hex digest of a raw session token. Only the
// digest is stored server side.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Handler serves the GitHub OAuth routes on the root mux.
type Handler struct {
	ClientID     string
	ClientSecret string
	// OAuthBaseURL defaults to https://github.com; tests point it at a fake.
	OAuthBaseURL string
	// APIBaseURL defaults to https://api.github.com; tests point it at a fake.
	APIBaseURL string
	// FrontendOrigin is the trusted dashboard origin for redirect_to.
	FrontendOrigin string
	// AdminLogin is the GitHub login allowed to claim the seeded default
	// workspace on first sign in.
	AdminLogin string
	Store      store.Store
	Now        func() time.Time
	HTTPClient *http.Client
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *Handler) client() *http.Client {
	if h.HTTPClient != nil {
		return h.HTTPClient
	}
	return http.DefaultClient
}

func (h *Handler) oauthBase() string {
	if h.OAuthBaseURL != "" {
		return strings.TrimSuffix(h.OAuthBaseURL, "/")
	}
	return "https://github.com"
}

func (h *Handler) apiBase() string {
	if h.APIBaseURL != "" {
		return strings.TrimSuffix(h.APIBaseURL, "/")
	}
	return "https://api.github.com"
}

// defaultFrontend is the trusted dashboard origin.
func (h *Handler) defaultFrontend() string {
	base := h.FrontendOrigin
	if base == "" {
		base = "http://localhost:3000"
	}
	return strings.TrimSuffix(base, "/")
}

// trustedRedirect reports whether redirect stays on the dashboard origin.
func trustedRedirect(base, redirect string) bool {
	return redirect == base || strings.HasPrefix(redirect, base+"/")
}

// signState builds the stateless signed state token. Format:
// <unix_ts>.<nonce>.<base64url(redirect_to)>.<hex_hmac>.
func (h *Handler) signState(ts int64, nonce, redirect string) string {
	b64 := base64.RawURLEncoding.EncodeToString([]byte(redirect))
	head := strconv.FormatInt(ts, 10) + "." + nonce + "." + b64
	return head + "." + h.hmacHex(head)
}

// verifyState checks the state HMAC and freshness, and returns the redirect.
func (h *Handler) verifyState(state string) (string, bool) {
	parts := strings.Split(state, ".")
	if len(parts) != 4 {
		return "", false
	}
	head := parts[0] + "." + parts[1] + "." + parts[2]
	if !hmac.Equal([]byte(h.hmacHex(head)), []byte(parts[3])) {
		return "", false
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", false
	}
	if delta := h.now().Sub(time.Unix(ts, 0)); delta > stateTTL || delta < -stateTTL {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func (h *Handler) hmacHex(msg string) string {
	mac := hmac.New(sha256.New, []byte(h.ClientSecret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// Register mounts the OAuth routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/github/start", h.start)
	mux.HandleFunc("GET /auth/github/callback", h.callback)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	base := h.defaultFrontend()
	redirect := strings.TrimSpace(r.URL.Query().Get("redirect_to"))
	if redirect == "" {
		redirect = base + "/auth/callback"
	}
	if !trustedRedirect(base, redirect) {
		httpx.Error(w, http.StatusBadRequest, "invalid redirect_to")
		return
	}
	state := h.signState(h.now().Unix(), id.Token(16), redirect)
	target := h.oauthBase() + "/login/oauth/authorize?client_id=" + url.QueryEscape(h.ClientID) +
		"&scope=read:user,user:email&state=" + url.QueryEscape(state)
	http.Redirect(w, r, target, http.StatusFound)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

type ghUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type ghEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	redirect, ok := h.verifyState(r.URL.Query().Get("state"))
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid state")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		httpx.Error(w, http.StatusBadRequest, "code is required")
		return
	}

	accessToken, err := h.exchangeCode(r.Context(), code)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "github oauth exchange failed")
		return
	}
	user, err := h.fetchUser(r.Context(), accessToken)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "github user lookup failed")
		return
	}

	claimDefault := strings.EqualFold(user.Login, h.AdminLogin)
	u, _, err := h.Store.UpsertGitHubUser(r.Context(), user.ID, user.Login, user.Name, user.Email, user.AvatarURL, claimDefault)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create user")
		return
	}

	raw := SessionTokenPrefix + id.Token(48)
	sess := &model.Session{
		TokenHash:   HashToken(raw),
		UserID:      u.ID,
		WorkspaceID: u.DefaultWorkspaceID,
		ExpiresAt:   h.now().Add(SessionTTL),
		CreatedAt:   h.now(),
	}
	if err := h.Store.CreateSession(r.Context(), sess); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create session")
		return
	}
	http.Redirect(w, r, redirect+"#session="+raw, http.StatusFound)
}

// exchangeCode trades the OAuth code for an access token.
func (h *Handler) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", h.ClientID)
	form.Set("client_secret", h.ClientSecret)
	form.Set("code", code)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.oauthBase()+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange status %d", resp.StatusCode)
	}
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return tr.AccessToken, nil
}

// fetchUser loads the GitHub profile, filling a missing email from
// /user/emails.
func (h *Handler) fetchUser(ctx context.Context, accessToken string) (*ghUser, error) {
	var user ghUser
	if err := h.getJSON(ctx, accessToken, "/user", &user); err != nil {
		return nil, err
	}
	if user.Email == "" {
		var emails []ghEmail
		if err := h.getJSON(ctx, accessToken, "/user/emails", &emails); err == nil {
			user.Email = pickEmail(emails)
		}
	}
	return &user, nil
}

func (h *Handler) getJSON(ctx context.Context, accessToken, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.apiBase()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := h.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github api %s status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// pickEmail prefers the primary verified address.
func pickEmail(emails []ghEmail) string {
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}
