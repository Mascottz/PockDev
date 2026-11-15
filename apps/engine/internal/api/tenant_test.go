package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/auth"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

// tenantEnv builds two signed in users with their own workspaces and session
// tokens.
type tenantEnv struct {
	a          *API
	st         *memstore.Mem
	h          http.Handler
	rawA       string
	rawB       string
	wspA       string
	wspB       string
	userA, userB string
}

func tenantSetup(t *testing.T) *tenantEnv {
	t.Helper()
	st := memstore.New()
	userA, wspA, err := st.UpsertGitHubUser(context.Background(), 2001, "alice", "Alice", "alice@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	userB, wspB, err := st.UpsertGitHubUser(context.Background(), 2002, "bob", "Bob", "bob@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	rawA := "pds_" + strings.Repeat("a", 48)
	rawB := "pds_" + strings.Repeat("b", 48)
	now := time.Now()
	sessions := map[string]struct{ u, w string }{
		rawA: {userA.ID, wspA.ID},
		rawB: {userB.ID, wspB.ID},
	}
	for raw, pair := range sessions {
		if err := st.CreateSession(context.Background(), &model.Session{
			TokenHash:   auth.HashToken(raw),
			UserID:      pair.u,
			WorkspaceID: pair.w,
			ExpiresAt:   now.Add(24 * time.Hour),
			CreatedAt:   now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	a := &API{Store: st, StartedAt: now, APIToken: ""}
	mux := http.NewServeMux()
	a.Register(mux)
	return &tenantEnv{
		a: a, st: st, h: mux,
		rawA: rawA, rawB: rawB,
		wspA: wspA.ID, wspB: wspB.ID,
		userA: userA.ID, userB: userB.ID,
	}
}

func TestMultiTenantIsolation(t *testing.T) {
	e := tenantSetup(t)

	// Both users create a project with the exact same slug.
	rec := do(t, e.h, http.MethodPost, "/api/projects", e.rawA, `{"name": "Orders API", "slug": "orders-api"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("A create: %d %s", rec.Code, rec.Body.String())
	}
	var prjA map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &prjA)

	rec = do(t, e.h, http.MethodPost, "/api/projects", e.rawB, `{"name": "Orders API", "slug": "orders-api"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("B create same slug in own workspace: %d %s", rec.Code, rec.Body.String())
	}
	var prjB map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &prjB)

	if prjA["id"] == prjB["id"] {
		t.Fatal("projects must be distinct")
	}
	if prjA["workspaceId"] != e.wspA || prjB["workspaceId"] != e.wspB {
		t.Fatalf("workspaces %v %v", prjA["workspaceId"], prjB["workspaceId"])
	}

	// Each list shows exactly one project: their own.
	rec = do(t, e.h, http.MethodGet, "/api/projects", e.rawA, "")
	var listA []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &listA)
	if len(listA) != 1 || listA[0]["id"] != prjA["id"] {
		t.Fatalf("A list %s", rec.Body.String())
	}
	rec = do(t, e.h, http.MethodGet, "/api/projects", e.rawB, "")
	var listB []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &listB)
	if len(listB) != 1 || listB[0]["id"] != prjB["id"] {
		t.Fatalf("B list %s", rec.Body.String())
	}

	// B accumulates extra rows: two paused deliveries and one open incident.
	now := time.Now()
	_ = e.st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d_b1", ProjectID: prjB["id"].(string), Status: model.DeliveryPaused, CreatedAt: now,
	})
	_ = e.st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d_b2", ProjectID: prjB["id"].(string), Status: model.DeliveryPaused, CreatedAt: now,
	})
	_ = e.st.CreateIncident(context.Background(), &model.Incident{
		ID: "i_b1", ProjectID: prjB["id"].(string), Title: "boom", Status: model.IncidentOpen,
		LastSeenAt: now, CreatedAt: now,
	})
	_ = e.st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "d_a1", ProjectID: prjA["id"].(string), Status: model.DeliveryPaused, CreatedAt: now,
	})
	_ = e.st.CreateIncident(context.Background(), &model.Incident{
		ID: "i_a1", ProjectID: prjA["id"].(string), Title: "boom A", Status: model.IncidentOpen,
		LastSeenAt: now, CreatedAt: now,
	})

	// A sees only A rows in listings.
	rec = do(t, e.h, http.MethodGet, "/api/deliveries", e.rawA, "")
	var dels []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &dels)
	if len(dels) != 1 || dels[0]["id"] != "d_a1" {
		t.Fatalf("A deliveries %s", rec.Body.String())
	}
	rec = do(t, e.h, http.MethodGet, "/api/incidents", e.rawA, "")
	var incs []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &incs)
	if len(incs) != 1 || incs[0]["id"] != "i_a1" {
		t.Fatalf("A incidents %s", rec.Body.String())
	}

	// A cannot touch B rows: every probe reports 404 so IDs leak nothing.
	rec = do(t, e.h, http.MethodGet, "/api/deliveries?projectId="+prjB["id"].(string), e.rawA, "")
	var foreign []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &foreign)
	if rec.Code != http.StatusOK || len(foreign) != 0 {
		t.Fatalf("A querying B project deliveries: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, e.h, http.MethodPost, "/api/deliveries/d_b1/replay", e.rawA, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("A replay B delivery: %d", rec.Code)
	}
	if rec := do(t, e.h, http.MethodPost, "/api/incidents/i_b1/action", e.rawA, `{"action": "mute"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("A action B incident: %d", rec.Code)
	}
	if rec := do(t, e.h, http.MethodPost, "/api/incidents/i_b1/action", e.rawA, `{"action": "resolve"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("A resolve B incident: %d", rec.Code)
	}

	// B cannot touch A rows either.
	if rec := do(t, e.h, http.MethodPost, "/api/deliveries/d_a1/replay", e.rawB, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("B replay A delivery: %d", rec.Code)
	}
	if rec := do(t, e.h, http.MethodPost, "/api/incidents/i_a1/action", e.rawB, `{"action": "mute"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("B action A incident: %d", rec.Code)
	}

	// Same IDOR wall for foreign project ids.
	if rec := do(t, e.h, http.MethodPost, "/api/projects", e.rawA, `{"name": "Dup", "slug": "orders-api"}`); rec.Code != http.StatusConflict {
		t.Fatalf("A duplicate slug in own workspace: %d", rec.Code)
	}

	// Overview counts stay strictly per workspace.
	rec = do(t, e.h, http.MethodGet, "/api/overview", e.rawA, "")
	var oA map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &oA)
	if oA["projectsCount"].(float64) != 1 || oA["pausedDeliveries"].(float64) != 1 || oA["openIncidents"].(float64) != 1 {
		t.Fatalf("A overview %s", rec.Body.String())
	}
	rec = do(t, e.h, http.MethodGet, "/api/overview", e.rawB, "")
	var oB map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &oB)
	if oB["projectsCount"].(float64) != 1 || oB["pausedDeliveries"].(float64) != 2 || oB["openIncidents"].(float64) != 1 {
		t.Fatalf("B overview %s", rec.Body.String())
	}

	// Cross workspace header membership is refused with 403.
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+e.rawA)
	req.Header.Set("X-PockDev-Workspace", e.wspB)
	rec2 := httptest.NewRecorder()
	e.h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("A claiming B workspace: %d", rec2.Code)
	}

	// The header workspace switch works for a workspace you belong to.
	req = httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+e.rawA)
	req.Header.Set("X-PockDev-Workspace", e.wspA)
	rec2 = httptest.NewRecorder()
	e.h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("A claiming A workspace: %d", rec2.Code)
	}
}
