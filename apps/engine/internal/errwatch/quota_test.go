package errwatch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

// TestMonthlyEventQuota fills the free plan monthly cap and expects 429 with
// the contract JSON body, and still accepts events under the cap.
func TestMonthlyEventQuota(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st := memstore.New()
	h := &Ingest{
		Store:          st,
		SpikeThreshold: 1,
		Window:         5 * time.Minute,
		Now:            func() time.Time { return now },
		OnIncident:     func(string) {},
	}
	proj := &model.Project{
		ID: "prj_e", Name: "Errs", Slug: "errs",
		EndpointKey: "ep_e", SDKKey: "sdk_e",
		Status: "active", CreatedAt: now,
	}
	if err := st.CreateProject(context.Background(), proj); err != nil {
		t.Fatal(err)
	}

	// Under the cap the ingest succeeds.
	req := httptest.NewRequest(http.MethodPost, "/v1/errors", strings.NewReader(`{"message": "boom"}`))
	req.Header.Set("X-PockDev-Key", "sdk_e")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("errors ingest under cap: %d %s", rec.Code, rec.Body.String())
	}

	// Fill the monthly cap with deliveries; error events share the budget.
	for i := 0; i < 5000; i++ {
		if err := st.CreateDelivery(context.Background(), &model.Delivery{
			ID: "dlv_e_" + strconv.Itoa(i), ProjectID: proj.ID, Status: model.DeliveryFailed,
			CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/errors", strings.NewReader(`{"message": "boom"}`))
	req.Header.Set("X-PockDev-Key", "sdk_e")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("errors ingest at cap: %d %s", rec.Code, rec.Body.String())
	}
	var errBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody["error"] != "monthly event quota reached for workspace" || errBody["code"] != "quota_exceeded" {
		t.Fatalf("body %s", rec.Body.String())
	}
}
