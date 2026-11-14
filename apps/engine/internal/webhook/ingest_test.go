package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

func setup(t *testing.T) (*Ingest, *memstore.Mem, *blob.Local) {
	t.Helper()
	st := memstore.New()
	local, err := blob.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := &Ingest{Store: st, Blobs: local, MaxBody: 1024}
	return h, st, local
}

func seedProject(t *testing.T, st *memstore.Mem) *model.Project {
	t.Helper()
	p := &model.Project{
		ID: "prj_1", Name: "orders-api", Slug: "orders-api",
		EndpointKey: "ek_1", SDKKey: "pk_1", Status: "active",
		CreatedAt: time.Now(),
	}
	if err := st.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIngestCatchStoreRespond(t *testing.T) {
	h, st, _ := setup(t)
	seedProject(t, st)

	body := `{"order": {"id": 7, "rider": {"phone": "+234"}}}`
	req := httptest.NewRequest(http.MethodPost, "/in/ek_1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Header", "abc")
	req.SetPathValue("endpointKey", "ek_1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Received || !strings.HasPrefix(resp.DeliveryID, "dlv_") {
		t.Fatalf("bad receipt: %s", rec.Body.String())
	}

	d, err := st.GetDelivery(context.Background(), resp.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != model.DeliveryRetrying {
		t.Fatalf("status %s", d.Status)
	}
	if d.PayloadPreview == "" || len(d.PayloadKeys) != 4 {
		t.Fatalf("preview/keys: %+v", d)
	}
	if d.Headers["x-test-header"] != "abc" {
		t.Fatalf("headers: %v", d.Headers)
	}
	if d.StorageKey == "" {
		t.Fatal("storage key missing")
	}
}

func TestIngestPersistsBody(t *testing.T) {
	h, st, local := setup(t)
	seedProject(t, st)
	body := `{"hello": "world"}`

	req := httptest.NewRequest(http.MethodPost, "/in/ek_1", strings.NewReader(body))
	req.SetPathValue("endpointKey", "ek_1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	d, err := st.GetDelivery(context.Background(), resp.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := local.Get(context.Background(), d.StorageKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("stored body %q", got)
	}
	if _, err := os.Stat(filepath.Join(local.Base(), d.StorageKey)); err != nil {
		t.Fatalf("body not on disk: %v", err)
	}
}

func TestIngestUnknownEndpoint(t *testing.T) {
	h, st, _ := setup(t)
	seedProject(t, st)
	req := httptest.NewRequest(http.MethodPost, "/in/nope", strings.NewReader("{}"))
	req.SetPathValue("endpointKey", "nope")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestIngestTooLarge(t *testing.T) {
	h, st, _ := setup(t)
	seedProject(t, st)
	big := strings.Repeat("x", 2048)
	req := httptest.NewRequest(http.MethodPost, "/in/ek_1", strings.NewReader(big))
	req.SetPathValue("endpointKey", "ek_1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestDeadLetterFilesIncidentAndDrift(t *testing.T) {
	st := memstore.New()
	seedProject(t, st)
	now := time.Now()

	// One healthy delivery carrying the old shape.
	healthy := &model.Delivery{
		ID: "dlv_ok", ProjectID: "prj_1", Status: model.DeliveryDelivered,
		PayloadKeys: []string{"order", "order.rider.phone"}, CreatedAt: now.Add(-time.Hour),
	}
	if err := st.CreateDelivery(context.Background(), healthy); err != nil {
		t.Fatal(err)
	}

	var triggered []string
	h := &DeadLetterHandler{
		Store:   st,
		Trigger: func(incID string) { triggered = append(triggered, incID) },
		Now:     func() time.Time { return now },
	}

	bad := &model.Delivery{
		ID: "dlv_bad", ProjectID: "prj_1", Status: model.DeliveryPaused,
		PayloadKeys: []string{"order", "order.rider.phoneNumber"}, CreatedAt: now,
	}
	if err := st.CreateDelivery(context.Background(), bad); err != nil {
		t.Fatal(err)
	}
	h.Handle(context.Background(), bad)

	updated, err := st.GetDelivery(context.Background(), "dlv_bad")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.SchemaDriftNote, "looks renamed") {
		t.Fatalf("drift note %q", updated.SchemaDriftNote)
	}
	if updated.LinkedIncidentID == "" {
		t.Fatal("delivery not linked to incident")
	}
	if len(triggered) != 1 {
		t.Fatalf("triggered %v", triggered)
	}

	inc, err := st.GetIncident(context.Background(), updated.LinkedIncidentID)
	if err != nil {
		t.Fatal(err)
	}
	if inc.Origin != model.OriginWebhookDeadLetter || inc.PausedCount != 1 {
		t.Fatalf("incident %+v", inc)
	}

	// A second dead letter reuses the same incident and does not re-alert.
	bad2 := &model.Delivery{
		ID: "dlv_bad2", ProjectID: "prj_1", Status: model.DeliveryPaused,
		PayloadKeys: []string{"order"}, CreatedAt: now,
	}
	if err := st.CreateDelivery(context.Background(), bad2); err != nil {
		t.Fatal(err)
	}
	h.Handle(context.Background(), bad2)
	if len(triggered) != 1 {
		t.Fatalf("should not re-trigger: %v", triggered)
	}
	inc2, _ := st.GetIncident(context.Background(), inc.ID)
	if inc2.PausedCount != 2 {
		t.Fatalf("pausedCount %d", inc2.PausedCount)
	}
}
