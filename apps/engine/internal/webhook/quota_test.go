package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

// TestMonthlyEventQuota fills the free plan monthly cap and expects 429 with
// the contract JSON body.
func TestMonthlyEventQuota(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st := memstore.New()
	blobs, err := blob.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := &Ingest{
		Store:   st,
		Blobs:   blobs,
		MaxBody: 1 << 20,
		Now:     func() time.Time { return now },
	}
	mux := http.NewServeMux()
	mux.Handle("POST /in/{endpointKey}", h)

	proj := &model.Project{
		ID: "prj_q", Name: "Quota", Slug: "quota",
		EndpointKey: "ep_quota", SDKKey: "sdk_quota",
		Status: "active", CreatedAt: now,
	}
	if err := st.CreateProject(context.Background(), proj); err != nil {
		t.Fatal(err)
	}

	// 4999 events in the current month: the 5000th still lands.
	for i := 0; i < 4999; i++ {
		if err := st.CreateDelivery(context.Background(), &model.Delivery{
			ID: "dlv_fill_" + strconv.Itoa(i), ProjectID: proj.ID, Status: model.DeliveryFailed,
			CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/in/ep_quota", strings.NewReader(`{"type":"order.created","data":{"id":"o1"}}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingest at 4999: %d %s", rec.Code, rec.Body.String())
	}

	// 5000 events: quota reached.
	req = httptest.NewRequest(http.MethodPost, "/in/ep_quota", strings.NewReader(`{"type":"x","data":{}}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("ingest at 5000: %d %s", rec.Code, rec.Body.String())
	}
	var errBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody["error"] != "monthly event quota reached for workspace" || errBody["code"] != "quota_exceeded" {
		t.Fatalf("body %s", rec.Body.String())
	}
}

// TestMonthlyEventQuotaAcrossChannels counts deliveries and error events in
// the same monthly budget.
func TestMonthlyEventQuotaAcrossChannels(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st := memstore.New()
	blobs, err := blob.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := &Ingest{
		Store:   st,
		Blobs:   blobs,
		MaxBody: 1 << 20,
		Now:     func() time.Time { return now },
	}
	mux := http.NewServeMux()
	mux.Handle("POST /in/{endpointKey}", h)
	proj := &model.Project{
		ID: "prj_c", Name: "C", Slug: "c",
		EndpointKey: "ep_c", SDKKey: "sdk_c",
		Status: "active", CreatedAt: now,
	}
	if err := st.CreateProject(context.Background(), proj); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3000; i++ {
		if err := st.CreateDelivery(context.Background(), &model.Delivery{
			ID: "dlv_c_" + strconv.Itoa(i), ProjectID: proj.ID, Status: model.DeliveryFailed,
			CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2000; i++ {
		if _, err := st.RecordErrorEvent(context.Background(), &model.ErrorEvent{
			ID: "err_c_" + strconv.Itoa(i), ProjectID: proj.ID, Message: "boom",
			Fingerprint: "fp_" + strconv.Itoa(i), CreatedAt: now,
		}, 5*time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/in/ep_c", strings.NewReader(`{"type":"x","data":{}}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("combined budget: %d %s", rec.Code, rec.Body.String())
	}
}
