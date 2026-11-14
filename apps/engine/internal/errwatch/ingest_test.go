package errwatch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

func seedProject(t *testing.T, st *memstore.Mem) {
	t.Helper()
	p := &model.Project{
		ID: "prj_1", Name: "orders-api", Slug: "orders-api",
		EndpointKey: "ek", SDKKey: "pk_test", Status: "active", CreatedAt: time.Now(),
	}
	if err := st.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func post(t *testing.T, h *Ingest, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/errors", strings.NewReader(body))
	if key != "" {
		req.Header.Set("X-PockDev-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestErrorIngestAuth(t *testing.T) {
	st := memstore.New()
	seedProject(t, st)
	h := &Ingest{Store: st, SpikeThreshold: 5}
	if rec := post(t, h, "", `{"message":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no key: %d", rec.Code)
	}
	if rec := post(t, h, "pk_wrong", `{"message":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad key: %d", rec.Code)
	}
}

func TestErrorIngestGroupsAndSpikes(t *testing.T) {
	st := memstore.New()
	seedProject(t, st)
	var triggered []string
	h := &Ingest{
		Store: st, SpikeThreshold: 3,
		OnIncident: func(incID string) { triggered = append(triggered, incID) },
	}

	for i := 0; i < 3; i++ {
		body := `{"message": "Cannot read rider.phone of undefined", "filePath": "src/orders.ts", "line": 48, "stack": "at orders.ts:48"}`
		rec := post(t, h, "pk_test", body)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("post %d: %s", rec.Code, rec.Body.String())
		}
	}
	if len(triggered) != 1 {
		t.Fatalf("expected one incident trigger, got %v", triggered)
	}

	// All three land in the same group.
	inc, err := st.GetIncident(context.Background(), triggered[0])
	if err != nil {
		t.Fatal(err)
	}
	if inc.Origin != model.OriginErrorSpike || inc.FilePath != "src/orders.ts" {
		t.Fatalf("incident %+v", inc)
	}
	g, err := st.GetErrorGroup(context.Background(), inc.OriginRef)
	if err != nil {
		t.Fatal(err)
	}
	if g.EventCount != 3 || g.IncidentID != inc.ID {
		t.Fatalf("group %+v", g)
	}
}

func TestErrorIngestLinksDelivery(t *testing.T) {
	st := memstore.New()
	seedProject(t, st)
	d := &model.Delivery{
		ID: "dlv_1", ProjectID: "prj_1", Status: model.DeliveryPaused,
		PayloadKeys: []string{"a"}, CreatedAt: time.Now(),
	}
	if err := st.CreateDelivery(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	var triggered []string
	h := &Ingest{
		Store: st, SpikeThreshold: 5,
		OnIncident: func(incID string) { triggered = append(triggered, incID) },
	}

	body := `{"message": "boom", "stack": "at orders (src/orders.ts:3)", "filePath": "src/orders.ts", "line": 3, "deliveryId": "dlv_1"}`
	if rec := post(t, h, "pk_test", body); rec.Code != http.StatusAccepted {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if len(triggered) != 1 {
		t.Fatalf("expected incident for crash on paused delivery, got %v", triggered)
	}
	updated, err := st.GetDelivery(context.Background(), "dlv_1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.LinkedIncidentID != triggered[0] {
		t.Fatalf("linked %q want %q", updated.LinkedIncidentID, triggered[0])
	}
	ev, err := st.ErrorEventByDelivery(context.Background(), "dlv_1")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Stack == "" {
		t.Fatalf("stack not stored: %+v", ev)
	}
}

func TestFingerprintStableAcrossNumbers(t *testing.T) {
	a := Fingerprint("prj", "src/a.ts", 1, "user 12345 failed")
	b := Fingerprint("prj", "src/a.ts", 9, "user 67890 failed")
	if a != b {
		t.Fatalf("fingerprints should match: %s vs %s", a, b)
	}
	c := Fingerprint("prj", "src/b.ts", 1, "user 12345 failed")
	if a == c {
		t.Fatal("different files should not match")
	}
}

func TestErrorIngestRequiresMessage(t *testing.T) {
	st := memstore.New()
	seedProject(t, st)
	h := &Ingest{Store: st, SpikeThreshold: 5}
	rec := post(t, h, "pk_test", `{"filePath": "x"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}
