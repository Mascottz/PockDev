package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

const testToken = "test-token"

func setupAPI(t *testing.T) (*API, *memstore.Mem, *agent.Agent) {
	t.Helper()
	st := memstore.New()
	pocketAgent := &agent.Agent{Store: st, MuteFor: time.Hour}
	a := &API{
		Store:     st,
		Agent:     pocketAgent,
		APIToken:  testToken,
		StartedAt: time.Now().Add(-90 * time.Second),
	}
	return a, st, pocketAgent
}

func do(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func mounted(a *API) http.Handler {
	mux := http.NewServeMux()
	a.Register(mux)
	return mux
}

func TestHealthz(t *testing.T) {
	a, _, _ := setupAPI(t)
	rec := do(t, mounted(a), http.MethodGet, "/healthz", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("body %s", rec.Body.String())
	}
	if _, ok := out["uptimeSeconds"].(float64); !ok {
		t.Fatalf("uptimeSeconds missing: %s", rec.Body.String())
	}
}

func TestAPIAuthRequired(t *testing.T) {
	a, _, _ := setupAPI(t)
	h := mounted(a)
	for _, path := range []string{"/api/overview", "/api/projects", "/api/deliveries", "/api/incidents"} {
		rec := do(t, h, http.MethodGet, path, "wrong", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", path, rec.Code)
		}
	}
}

func TestOverview(t *testing.T) {
	a, st, _ := setupAPI(t)
	_ = st.CreateProject(context.Background(), &model.Project{
		ID: "prj_1", Name: "n", Slug: "n", EndpointKey: "e", SDKKey: "s",
		Status: "active", CreatedAt: time.Now(),
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d1", ProjectID: "prj_1", Status: model.DeliveryDelivered, CreatedAt: time.Now(),
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d2", ProjectID: "prj_1", Status: model.DeliveryPaused, CreatedAt: time.Now(),
	})
	_ = st.CreateIncident(context.Background(), &model.Incident{
		ID: "i1", ProjectID: "prj_1", Title: "t", Status: model.IncidentOpen,
		LastSeenAt: time.Now(), CreatedAt: time.Now(),
	})

	rec := do(t, mounted(a), http.MethodGet, "/api/overview", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"projectsCount", "openIncidents", "pausedDeliveries", "caughtToday", "healthyRate"} {
		if _, ok := out[key]; !ok {
			t.Fatalf("missing %s in %s", key, rec.Body.String())
		}
	}
	if out["projectsCount"].(float64) != 1 || out["pausedDeliveries"].(float64) != 1 {
		t.Fatalf("overview %s", rec.Body.String())
	}
	if out["healthyRate"].(float64) != 50 {
		t.Fatalf("healthyRate %v", out["healthyRate"])
	}
}

func TestProjectLifecycle(t *testing.T) {
	a, _, _ := setupAPI(t)
	h := mounted(a)

	rec := do(t, h, http.MethodPost, "/api/projects", testToken,
		`{"name": "Orders API", "targetUrl": "https://example.com/hook", "githubRepo": "acme/orders", "whatsappPhone": "+234801"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var p map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "name", "slug", "endpointKey", "sdkKey", "targetUrl", "githubRepo", "whatsappPhone", "status", "createdAt"} {
		if _, ok := p[key]; !ok {
			t.Fatalf("missing %s in %s", key, rec.Body.String())
		}
	}
	if p["slug"] != "orders-api" {
		t.Fatalf("slug %v", p["slug"])
	}
	if !strings.HasPrefix(p["sdkKey"].(string), "pk_") {
		t.Fatalf("sdkKey %v", p["sdkKey"])
	}

	rec = do(t, h, http.MethodGet, "/api/projects", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list %s", rec.Body.String())
	}

	rec = do(t, h, http.MethodPost, "/api/projects", testToken, `{"name": ""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name: %d", rec.Code)
	}
}

func TestDeliveriesAndReplay(t *testing.T) {
	a, st, _ := setupAPI(t)
	h := mounted(a)
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d1", ProjectID: "prj_1", Status: model.DeliveryPaused,
		AttemptCount: 8, PayloadPreview: `{"a":1}`, CreatedAt: time.Now(),
	})

	rec := do(t, h, http.MethodGet, "/api/deliveries?projectId=prj_1&status=paused", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list %s", rec.Body.String())
	}
	for _, key := range []string{"id", "projectId", "status", "httpStatus", "attemptCount", "schemaDriftNote", "payloadPreview", "linkedIncidentId", "createdAt"} {
		if _, ok := list[0][key]; !ok {
			t.Fatalf("missing %s in %s", key, rec.Body.String())
		}
	}

	rec = do(t, h, http.MethodPost, "/api/deliveries/d1/replay", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("replay: %d %s", rec.Code, rec.Body.String())
	}
	var d map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d["status"] != "retrying" || d["attemptCount"].(float64) != 0 {
		t.Fatalf("replayed %s", rec.Body.String())
	}

	rec = do(t, h, http.MethodPost, "/api/deliveries/nope/replay", testToken, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing delivery: %d", rec.Code)
	}
}

func TestIncidentActions(t *testing.T) {
	a, st, _ := setupAPI(t)
	h := mounted(a)
	now := time.Now()
	_ = st.CreateIncident(context.Background(), &model.Incident{
		ID: "i1", ProjectID: "prj_1", Title: "Crash in orders", Status: model.IncidentOpen,
		LastSeenAt: now, CreatedAt: now,
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d1", ProjectID: "prj_1", Status: model.DeliveryPaused,
		LinkedIncidentID: "i1", CreatedAt: now,
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d2", ProjectID: "prj_1", Status: model.DeliveryPaused,
		LinkedIncidentID: "i1", CreatedAt: now,
	})

	rec := do(t, h, http.MethodGet, "/api/incidents?projectId=prj_1", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list %s", rec.Body.String())
	}
	for _, key := range []string{"id", "projectId", "title", "plainSummary", "filePath", "commitSha", "status", "pausedCount", "prUrl", "lastSeenAt"} {
		if _, ok := list[0][key]; !ok {
			t.Fatalf("missing %s in %s", key, rec.Body.String())
		}
	}

	rec = do(t, h, http.MethodPost, "/api/incidents/i1/action", testToken, `{"action": "replay_all"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("replay_all: %d %s", rec.Code, rec.Body.String())
	}
	var inc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &inc); err != nil {
		t.Fatal(err)
	}
	if inc["pausedCount"].(float64) != 0 {
		t.Fatalf("incident %s", rec.Body.String())
	}

	rec = do(t, h, http.MethodPost, "/api/incidents/i1/action", testToken, `{"action": "mute"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("mute: %d %s", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &inc)
	if inc["status"] != "muted" {
		t.Fatalf("incident %s", rec.Body.String())
	}

	rec = do(t, h, http.MethodPost, "/api/incidents/i1/action", testToken, `{"action": "resolve"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve: %d %s", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &inc)
	if inc["status"] != "resolved" {
		t.Fatalf("incident %s", rec.Body.String())
	}

	rec = do(t, h, http.MethodPost, "/api/incidents/i1/action", testToken, `{"action": "wat"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad action: %d", rec.Code)
	}
}

func TestMuteExpires(t *testing.T) {
	_, st, _ := setupAPI(t)
	now := time.Now()
	until := now.Add(-time.Minute)
	_ = st.CreateIncident(context.Background(), &model.Incident{
		ID: "i1", ProjectID: "prj_1", Title: "t", Status: model.IncidentMuted,
		MuteUntil: &until, LastSeenAt: now, CreatedAt: now,
	})
	n, err := st.ExpireMutes(context.Background(), now)
	if err != nil || n != 1 {
		t.Fatalf("expire: %v %d", err, n)
	}
	inc, _ := st.GetIncident(context.Background(), "i1")
	if inc.Status != model.IncidentOpen {
		t.Fatalf("status %s", inc.Status)
	}
}

func TestCORS(t *testing.T) {
	a, _, _ := setupAPI(t)
	mux := http.NewServeMux()
	a.Register(mux)
	h := httpx.CORS([]string{"http://localhost:3000"}, mux)

	req := httptest.NewRequest(http.MethodOptions, "/api/overview", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("cors header %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/overview", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("origin should not be allowed")
	}
}
