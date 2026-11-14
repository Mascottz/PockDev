// Package api serves the shared REST contract for the PockDev dashboard in
// apps/web (Session 2). Route names and JSON fields are frozen.
package api

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
	"github.com/Mascottz/PockDev/apps/engine/internal/worker"
)

// API holds the dashboard routes.
type API struct {
	Store     store.Store
	Agent     *agent.Agent
	APIToken  string
	StartedAt time.Time
	Now       func() time.Time
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
	mux.Handle("/api/", api)

	mux.HandleFunc("GET /healthz", a.healthz)
}

func (a *API) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *API) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if httpx.Bearer(r) != a.APIToken {
			httpx.Error(w, http.StatusUnauthorized, "invalid API token")
			return
		}
		next(w, r)
	}
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	up := int(a.now().Sub(a.StartedAt).Seconds())
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "uptimeSeconds": up})
}

func (a *API) overview(w http.ResponseWriter, r *http.Request) {
	o, err := a.Store.Overview(r.Context(), a.now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load overview")
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (a *API) listProjects(w http.ResponseWriter, r *http.Request) {
	list, err := a.Store.ListProjects(r.Context())
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
		slug = slugify(req.Name)
	}
	now := a.now()
	p := &model.Project{
		ID:            id.New("prj_", 16),
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
	f := model.DeliveryFilter{
		ProjectID: r.URL.Query().Get("projectId"),
		Status:    r.URL.Query().Get("status"),
	}
	list, err := a.Store.ListDeliveries(r.Context(), f)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load deliveries")
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (a *API) replayDelivery(w http.ResponseWriter, r *http.Request) {
	d, err := worker.Replay(r.Context(), a.Store, r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "delivery not found")
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (a *API) listIncidents(w http.ResponseWriter, r *http.Request) {
	f := model.IncidentFilter{ProjectID: r.URL.Query().Get("projectId")}
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
	var req incidentActionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	incidentID := r.PathValue("id")
	ctx := r.Context()

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

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugStrip.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "project"
	}
	return s
}
