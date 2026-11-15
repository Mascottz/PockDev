package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

func seed(t *testing.T, st *memstore.Mem, blobs *blob.Local, target string) *model.Project {
	t.Helper()
	p := &model.Project{
		ID: "prj_1", Name: "orders-api", Slug: "orders-api",
		EndpointKey: "ek", SDKKey: "pk", TargetURL: target,
		Status: "active", CreatedAt: time.Now(),
	}
	if err := st.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func addDelivery(t *testing.T, st *memstore.Mem, blobs *blob.Local, id string) *model.Delivery {
	t.Helper()
	d := &model.Delivery{
		ID: id, ProjectID: "prj_1", Status: model.DeliveryRetrying,
		StorageKey:    "deliveries/prj_1/" + id + ".body",
		Headers:       map[string]string{"x-test": "1"},
		NextAttemptAt: time.Now().Add(-time.Second),
		CreatedAt:     time.Now(),
	}
	if err := blobs.Put(context.Background(), d.StorageKey, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDelivery(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return d
}

func runOnce(t *testing.T, f *Forwarder, st *memstore.Mem) *model.Delivery {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	batch, err := st.ClaimDueDeliveries(ctx, time.Now(), time.Minute, 10)
	if err != nil || len(batch) == 0 {
		t.Fatalf("claim: %v %d", err, len(batch))
	}
	for i := range batch {
		f.handle(ctx, &batch[i])
	}
	d, err := st.GetDelivery(context.Background(), batch[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestForwardSuccess(t *testing.T) {
	var gotDeliveryID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDeliveryID = r.Header.Get("X-PockDev-Delivery-Id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	st := memstore.New()
	local, _ := blob.NewLocal(t.TempDir())
	seed(t, st, local, srv.URL)
	addDelivery(t, st, local, "dlv_1")

	f := &Forwarder{Store: st, Blobs: local, MaxAttempts: 3, RetryBase: time.Millisecond}
	d := runOnce(t, f, st)
	if d.Status != model.DeliveryDelivered {
		t.Fatalf("status %s err %s", d.Status, d.LastError)
	}
	if d.HTTPStatus != 200 || d.AttemptCount != 1 {
		t.Fatalf("delivery %+v", d)
	}
	if gotDeliveryID != "dlv_1" {
		t.Fatalf("header %q", gotDeliveryID)
	}
}

func TestForwardRetriesThenPauses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	st := memstore.New()
	local, _ := blob.NewLocal(t.TempDir())
	seed(t, st, local, srv.URL)
	addDelivery(t, st, local, "dlv_1")

	dead := make(chan string, 4)
	f := &Forwarder{
		Store: st, Blobs: local, MaxAttempts: 2, RetryBase: time.Millisecond,
		OnDeadLetter: func(ctx context.Context, d *model.Delivery) { dead <- d.ID },
	}

	d := runOnce(t, f, st)
	if d.Status != model.DeliveryRetrying || d.AttemptCount != 1 {
		t.Fatalf("after first: %+v", d)
	}
	time.Sleep(5 * time.Millisecond)
	d = runOnce(t, f, st)
	if d.Status != model.DeliveryPaused || d.AttemptCount != 2 {
		t.Fatalf("after second: %+v", d)
	}
	select {
	case id := <-dead:
		if id != "dlv_1" {
			t.Fatalf("dead letter %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("dead letter callback missing")
	}
}

func TestForwardPermanentFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	st := memstore.New()
	local, _ := blob.NewLocal(t.TempDir())
	seed(t, st, local, srv.URL)
	addDelivery(t, st, local, "dlv_1")

	f := &Forwarder{Store: st, Blobs: local, MaxAttempts: 5, RetryBase: time.Millisecond}
	d := runOnce(t, f, st)
	if d.Status != model.DeliveryFailed {
		t.Fatalf("status %s", d.Status)
	}
	if d.HTTPStatus != 403 {
		t.Fatalf("http %d", d.HTTPStatus)
	}
}

func TestReplayResetsDelivery(t *testing.T) {
	st := memstore.New()
	d := &model.Delivery{
		ID: "dlv_1", ProjectID: "prj_1", Status: model.DeliveryPaused,
		AttemptCount: 8, CreatedAt: time.Now(),
	}
	if err := st.CreateDelivery(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	got, err := Replay(context.Background(), st, "dlv_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.DeliveryRetrying || got.AttemptCount != 0 {
		t.Fatalf("replay %+v", got)
	}
}

func TestReplayAllCountsPaused(t *testing.T) {
	st := memstore.New()
	for _, s := range []string{model.DeliveryPaused, model.DeliveryPaused, model.DeliveryFailed, model.DeliveryDelivered} {
		d := &model.Delivery{
			ID: "dlv_" + s + time.Now().String(), ProjectID: "prj_1", Status: s,
			LinkedIncidentID: "inc_1", CreatedAt: time.Now(),
		}
		if err := st.CreateDelivery(context.Background(), d); err != nil {
			t.Fatal(err)
		}
	}
	n, err := ReplayAll(context.Background(), st, "inc_1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("replayed %d", n)
	}
}

func TestReplayProjectCountsPausedAndFailed(t *testing.T) {
	st := memstore.New()
	now := time.Now()
	entries := []struct {
		id, projectID, status string
	}{
		{"dlv_p1", "prj_1", model.DeliveryPaused},
		{"dlv_p2", "prj_1", model.DeliveryFailed},
		{"dlv_p3", "prj_1", model.DeliveryDelivered},
		{"dlv_other", "prj_2", model.DeliveryPaused},
	}
	for _, e := range entries {
		if err := st.CreateDelivery(context.Background(), &model.Delivery{
			ID: e.id, ProjectID: e.projectID, Status: e.status, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := ReplayProject(context.Background(), st, "prj_1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("replayed %d, want 2", n)
	}
	for _, id := range []string{"dlv_p1", "dlv_p2"} {
		d, _ := st.GetDelivery(context.Background(), id)
		if d.Status != model.DeliveryRetrying {
			t.Fatalf("%s status %s, want retrying", id, d.Status)
		}
	}
	other, _ := st.GetDelivery(context.Background(), "dlv_other")
	if other.Status != model.DeliveryPaused {
		t.Fatalf("other project delivery changed to %s", other.Status)
	}
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	f := &Forwarder{RetryBase: time.Second, MaxBackoff: 8 * time.Second}
	if got := f.backoff(1); got != time.Second {
		t.Fatalf("attempt 1: %v", got)
	}
	if got := f.backoff(3); got != 4*time.Second {
		t.Fatalf("attempt 3: %v", got)
	}
	if got := f.backoff(9); got != 8*time.Second {
		t.Fatalf("attempt 9: %v", got)
	}
}

func TestClaimLease(t *testing.T) {
	st := memstore.New()
	d := &model.Delivery{
		ID: "dlv_1", ProjectID: "prj_1", Status: model.DeliveryRetrying,
		NextAttemptAt: time.Now().Add(-time.Second), CreatedAt: time.Now(),
	}
	if err := st.CreateDelivery(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	batch, err := st.ClaimDueDeliveries(context.Background(), now, time.Minute, 10)
	if err != nil || len(batch) != 1 {
		t.Fatalf("claim 1: %v %d", err, len(batch))
	}
	batch2, err := st.ClaimDueDeliveries(context.Background(), now, time.Minute, 10)
	if err != nil || len(batch2) != 0 {
		t.Fatalf("claim 2 should be empty: %v %d", err, len(batch2))
	}
}
