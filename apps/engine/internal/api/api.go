// Package api serves the shared REST contract for the PockDev dashboard in
// apps/web (Session 2). Route names and JSON fields are frozen.
package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/auth"
	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
	"github.com/Mascottz/PockDev/apps/engine/internal/worker"
)

// API holds the dashboard routes.
type API struct {
	Store    store.Store
	Agent    *agent.Agent
	APIToken string
	// GitHubAuthConfigured reports whether GitHub OAuth sign in is enabled.
	GitHubAuthConfigured bool
	StartedAt            time.Time
	Now                  func() time.Time
}

// authInfo is the resolved auth context for one request.
type authInfo struct {
	// Mode is "session" for GitHub logins and "token" for API_TOKEN calls.
	Mode string
	// User is nil in token mode.
	User *model.User
	// WorkspaceID is the workspace every query must respect.
	WorkspaceID string
}

type authInfoKey struct{}

func withAuthInfo(r *http.Request, info *authInfo) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authInfoKey{}, info))
}

func authInfoFrom(r *http.Request) *authInfo {
	info, _ := r.Context().Value(authInfoKey{}).(*authInfo)
	if info == nil {
		return &authInfo{Mode: "token", WorkspaceID: store.DefaultWorkspaceID}
	}
	return info
}

// Register mounts every route on mux.
func (a *API) Register(mux *http.ServeMux) {
	api := http.NewServeMux()
	api.HandleFunc("GET /api/overview", a.auth(a.overview))
	api.HandleFunc("GET /api/projects", a.auth(a.listProjects))
	api.HandleFunc("POST /api/projects", a.auth(a.createProject))
	api.HandleFunc("GET /api/deliveries", a.auth(a.listDeliveries))
	api.HandleFunc("POST /api/deliveries/{id}/replay", a.auth(a.replayDelivery))
	api.HandleFunc("GET /api/incidents", a.auth(a.listIncidents))
	api.HandleFunc("POST /api/incidents/{id}/action", a.auth(a.incidentAction))
	api.HandleFunc("GET /api/me", a.auth(a.me))
	api.HandleFunc("POST /api/workspaces", a.auth(a.createWorkspace))
	api.HandleFunc("POST /api/auth/logout", a.logout)
	mux.Handle("/api/", api)

	mux.HandleFunc("GET /healthz", a.healthz)
}

func (a *API) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// auth resolves the caller into an auth context:
//
// Case 1: a Bearer pds_ session token resolves through Store.GetSession. The
// X-PockDev-Workspace header switches the active workspace after a membership
// check.
//
// Case 2: a Bearer token matching APIToken runs in token mode against the
// default workspace, or the header workspace for admin inspection.
//
// Anything else is rejected with 401.
func (a *API) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := httpx.Bearer(r)
		hdrWorkspace := strings.TrimSpace(r.Header.Get("X-PockDev-Workspace"))

		if strings.HasPrefix(token, auth.SessionTokenPrefix) {
			sess, user, err := a.Store.GetSession(r.Context(), auth.HashToken(token), a.now())
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}
			workspaceID := sess.WorkspaceID
			if hdrWorkspace != "" {
				ok, err := a.Store.UserHasWorkspace(r.Context(), user.ID, hdrWorkspace)
				if err != nil {
					httpx.Error(w, http.StatusInternalServerError, "could not load workspace")
					return
				}
				if !ok {
					httpx.Error(w, http.StatusForbidden, "forbidden")
					return
				}
				workspaceID = hdrWorkspace
			}
			next(w, withAuthInfo(r, &authInfo{Mode: "session", User: user, WorkspaceID: workspaceID}))
			return
		}

		if a.APIToken != "" && token == a.APIToken {
			workspaceID := store.DefaultWorkspaceID
			if hdrWorkspace != "" {
				workspaceID = hdrWorkspace
			}
			next(w, withAuthInfo(r, &authInfo{Mode: "token", WorkspaceID: workspaceID}))
			return
		}

		httpx.Error(w, http.StatusUnauthorized, "authentication required")
	}
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	up := int(a.now().Sub(a.StartedAt).Seconds())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":                  true,
		"uptimeSeconds":       up,
		"githubAuthConfigured": a.GitHubAuthConfigured,
	})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	wsp, err := a.Store.GetWorkspace(r.Context(), info.WorkspaceID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "workspace not found")
		return
	}
	var user any
	if info.User != nil {
		user = map[string]string{
			"id":          info.User.ID,
			"githubLogin": info.User.GitHubLogin,
			"name":        info.User.Name,
			"email":       info.User.Email,
			"avatarUrl":   info.User.AvatarURL,
		}
	}
	workspaces := []model.WorkspaceSummary{}
	if info.Mode == "session" {
		list, err := a.Store.ListUserWorkspaces(r.Context(), info.User.ID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "could not load workspaces")
			return
		}
		workspaces = list
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"mode":      info.Mode,
		"user":      user,
		"workspace": map[string]string{"id": wsp.ID, "name": wsp.Name, "slug": wsp.Slug},
		"workspaces": workspaces,
	})
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (a *API) createWorkspace(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	if info.Mode != "session" {
		httpx.Error(w, http.StatusForbidden, "session required")
		return
	}
	var req createWorkspaceRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = store.Slugify(name)
	}
	wsp, err := a.Store.CreateWorkspace(r.Context(), info.User.ID, name, slug)
	if err != nil {
		httpx.Error(w, http.StatusConflict, "workspace with this slug already exists")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{
		"id":   wsp.ID,
		"name": wsp.Name,
		"slug": wsp.Slug,
		"role": "owner",
	})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	token := httpx.Bearer(r)
	if strings.HasPrefix(token, auth.SessionTokenPrefix) {
		_ = a.Store.DeleteSession(r.Context(), auth.HashToken(token))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) overview(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	o, err := a.Store.Overview(r.Context(), info.WorkspaceID, a.now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load overview")
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (a *API) listProjects(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	list, err := a.Store.ListProjects(r.Context(), info.WorkspaceID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load projects")
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

type createProjectRequest struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	TargetURL     string `json:"targetUrl"`
	GitHubRepo    string `json:"githubRepo"`
	WhatsAppPhone string `json:"whatsappPhone"`
}

func (a *API) createProject(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	var req createProjectRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = store.Slugify(req.Name)
	}
	now := a.now()
	p := &model.Project{
		ID:            id.New("prj_", 16),
		WorkspaceID:   info.WorkspaceID,
		Name:          req.Name,
		Slug:          slug,
		EndpointKey:   id.Token(24),
		SDKKey:        "pk_" + id.Token(32),
		TargetURL:     strings.TrimSpace(req.TargetURL),
		GitHubRepo:    strings.TrimSpace(req.GitHubRepo),
		WhatsAppPhone: strings.TrimSpace(req.WhatsAppPhone),
		Status:        "active",
		CreatedAt:     now,
	}
	if err := a.Store.CreateProject(r.Context(), p); err != nil {
		httpx.Error(w, http.StatusConflict, "project with this slug or key already exists")
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (a *API) listDeliveries(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	f := model.DeliveryFilter{
		WorkspaceID: info.WorkspaceID,
		ProjectID:   r.URL.Query().Get("projectId"),
		Status:      r.URL.Query().Get("status"),
	}
	list, err := a.Store.ListDeliveries(r.Context(), f)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load deliveries")
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

// projectInWorkspace loads the delivery project and checks it belongs to the
// caller workspace. A miss reports 404 so cross workspace probing is
// impossible.
func (a *API) projectInWorkspace(r *http.Request, projectID, workspaceID string) bool {
	p, err := a.Store.GetProject(r.Context(), projectID)
	if err != nil {
		return false
	}
	pws := p.WorkspaceID
	if pws == "" {
		pws = store.DefaultWorkspaceID
	}
	return pws == workspaceID
}

func (a *API) replayDelivery(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	deliveryID := r.PathValue("id")
	d, err := a.Store.GetDelivery(r.Context(), deliveryID)
	if err != nil || !a.projectInWorkspace(r, d.ProjectID, info.WorkspaceID) {
		httpx.Error(w, http.StatusNotFound, "delivery not found")
		return
	}
	d, err = worker.Replay(r.Context(), a.Store, deliveryID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "delivery not found")
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (a *API) listIncidents(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	f := model.IncidentFilter{
		WorkspaceID: info.WorkspaceID,
		ProjectID:   r.URL.Query().Get("projectId"),
	}
	list, err := a.Store.ListIncidents(r.Context(), f)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load incidents")
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

type incidentActionRequest struct {
	Action string `json:"action"`
}

func (a *API) incidentAction(w http.ResponseWriter, r *http.Request) {
	info := authInfoFrom(r)
	var req incidentActionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	incidentID := r.PathValue("id")
	ctx := r.Context()

	inc, err := a.Store.GetIncident(ctx, incidentID)
	if err != nil || !a.projectInWorkspace(r, inc.ProjectID, info.WorkspaceID) {
		httpx.Error(w, http.StatusNotFound, "incident not found")
		return
	}

	switch req.Action {
	case "draft_pr":
		if a.Agent == nil {
			httpx.Error(w, http.StatusServiceUnavailable, "agent is not configured")
			return
		}
		inc, err := a.Agent.DraftPR(ctx, incidentID)
		if err != nil {
			httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		httpx.JSON(w, http.StatusOK, inc)
	case "replay_all":
		if a.Agent == nil {
			httpx.Error(w, http.StatusServiceUnavailable, "agent is not configured")
			return
		}
		inc, _, err := a.Agent.ReplayAll(ctx, incidentID)
		if err != nil {
			httpx.Error(w, http.StatusNotFound, "incident not found")
			return
		}
		httpx.JSON(w, http.StatusOK, inc)
	case "resolve":
		if a.Agent == nil {
			httpx.Error(w, http.StatusServiceUnavailable, "agent is not configured")
			return
		}
		inc, err := a.Agent.Resolve(ctx, incidentID)
		if err != nil {
			httpx.Error(w, http.StatusNotFound, "incident not found")
			return
		}
		httpx.JSON(w, http.StatusOK, inc)
	case "mute":
		if a.Agent == nil {
			httpx.Error(w, http.StatusServiceUnavailable, "agent is not configured")
			return
		}
		inc, err := a.Agent.Mute(ctx, incidentID)
		if err != nil {
			httpx.Error(w, http.StatusNotFound, "incident not found")
			return
		}
		httpx.JSON(w, http.StatusOK, inc)
	default:
		httpx.Error(w, http.StatusBadRequest, "action must be draft_pr, replay_all, resolve, or mute")
	}
}
