package worker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

func TestRetentionSweeperPrunesExpiredData(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	st := memstore.New()

	// A free workspace (default) and a pro workspace.
	userPro, wspPro, err := st.UpsertGitHubUser(ctx, 9001, "pro", "Pro", "pro@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpgradeWorkspacePro(ctx, wspPro.ID, "ref_p", "cus_p", "sub_p", "NGN", 1500000, now.Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	projFree := &model.Project{
		ID: "prj_free", Name: "Free", Slug: "free",
		EndpointKey: "ep_free", SDKKey: "sdk_free",
		WorkspaceID: store.DefaultWorkspaceID, Status: "active", CreatedAt: now,
	}
	projPro := &model.Project{
		ID: "prj_pro", Name: "Pro", Slug: "pro",
		EndpointKey: "ep_pro", SDKKey: "sdk_pro",
		WorkspaceID: wspPro.ID, Status: "active", CreatedAt: now,
	}
	for _, p := range []*model.Project{projFree, projPro} {
		if err := st.CreateProject(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	deliveries := []*model.Delivery{
		// Free workspace: 3 day retention.
		{ID: "dlv_old_delivered", ProjectID: projFree.ID, Status: model.DeliveryDelivered, StorageKey: "blobs/dlv_old_delivered.json", CreatedAt: now.Add(-4 * day)},
		{ID: "dlv_old_failed", ProjectID: projFree.ID, Status: model.DeliveryFailed, StorageKey: "blobs/missing.json", CreatedAt: now.Add(-10 * day)},
		{ID: "dlv_old_paused", ProjectID: projFree.ID, Status: model.DeliveryPaused, CreatedAt: now.Add(-10 * day)},
		{ID: "dlv_old_retrying", ProjectID: projFree.ID, Status: model.DeliveryRetrying, AttemptCount: 2, CreatedAt: now.Add(-10 * day)},
		{ID: "dlv_recent_delivered", ProjectID: projFree.ID, Status: model.DeliveryDelivered, CreatedAt: now.Add(-1 * day)},
		{ID: "dlv_open", ProjectID: projFree.ID, Status: "pending", CreatedAt: now.Add(-50 * day)},
		// Pro workspace: 30 day retention.
		{ID: "dlv_pro_mid", ProjectID: projPro.ID, Status: model.DeliveryDelivered, CreatedAt: now.Add(-10 * day)},
		{ID: "dlv_pro_old", ProjectID: projPro.ID, Status: model.DeliveryDelivered, StorageKey: "blobs/dlv_pro_old.json", CreatedAt: now.Add(-35 * day)},
	}
	for _, d := range deliveries {
		if err := st.CreateDelivery(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	// Distinct fingerprints make one group per event.
	groups := map[string]string{}
	for id, fp := range map[string]string{
		"err_old":    "fp_old",
		"err_recent": "fp_rec",
		"err_pro_mid": "fp_mid",
		"err_pro_old": "fp_pold",
	} {
		created := now.Add(-4 * day)
		switch id {
		case "err_recent":
			created = now.Add(-1 * day)
		case "err_pro_mid":
			created = now.Add(-10 * day)
		case "err_pro_old":
			created = now.Add(-35 * day)
		}
		projID := projFree.ID
		if id == "err_pro_mid" || id == "err_pro_old" {
			projID = projPro.ID
		}
		g, err := st.RecordErrorEvent(ctx, &model.ErrorEvent{
			ID: id, ProjectID: projID, Message: id,
			Fingerprint: fp, CreatedAt: created,
		}, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		groups[id] = g.ID
	}

	if err := st.CreateSession(ctx, &model.Session{
		TokenHash: "sess_dead", UserID: userPro.ID, WorkspaceID: wspPro.ID,
		ExpiresAt: now.Add(-time.Hour), CreatedAt: now.Add(-2 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(ctx, &model.Session{
		TokenHash: "sess_live", UserID: userPro.ID, WorkspaceID: wspPro.ID,
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Blob files: one present, the other intentionally missing.
	dir := t.TempDir()
	blobs, err := blob.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := blobs.Put(ctx, "blobs/dlv_old_delivered.json", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := blobs.Put(ctx, "blobs/dlv_pro_old.json", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}

	s := &RetentionSweeper{
		Store:    st,
		Blobs:    blobs,
		Interval: time.Hour,
		Now:      func() time.Time { return now },
		Logf:     func(string, ...any) {},
	}
	keys, nDeliv, nErr, nSess, err := s.Sweep(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if nDeliv != 3 || nErr != 2 || nSess != 1 {
		t.Fatalf("counts deliveries=%d errors=%d sessions=%d", nDeliv, nErr, nSess)
	}
	// All three pruned rows carried keys, including the intentionally
	// missing blob; deleting a missing blob is not an error.
	if len(keys) != 3 {
		t.Fatalf("storage keys %v", keys)
	}

	// Terminal pruned, paused and retrying kept, pending never touched.
	for _, id := range []string{"dlv_old_delivered", "dlv_old_failed", "dlv_pro_old"} {
		if _, err := st.GetDelivery(ctx, id); err == nil {
			t.Fatalf("%s must be pruned", id)
		}
	}
	for _, id := range []string{"dlv_old_paused", "dlv_old_retrying", "dlv_recent_delivered", "dlv_open", "dlv_pro_mid"} {
		if _, err := st.GetDelivery(ctx, id); err != nil {
			t.Fatalf("%s must survive: %v", id, err)
		}
	}

	// Error events follow the same 3d vs 30d windows.
	if _, err := st.LatestErrorEvent(ctx, groups["err_old"]); err == nil {
		t.Fatal("old free error event must be pruned")
	}
	if _, err := st.LatestErrorEvent(ctx, groups["err_pro_old"]); err == nil {
		t.Fatal("old pro error event must be pruned")
	}
	if _, err := st.LatestErrorEvent(ctx, groups["err_recent"]); err != nil {
		t.Fatalf("recent error event must survive: %v", err)
	}
	if _, err := st.LatestErrorEvent(ctx, groups["err_pro_mid"]); err != nil {
		t.Fatalf("10 day old pro error event must survive: %v", err)
	}

	// Expired session gone, live one kept.
	if _, _, err := st.GetSession(ctx, "sess_dead", now); err == nil {
		t.Fatal("expired session must be pruned")
	}
	if _, _, err := st.GetSession(ctx, "sess_live", now); err != nil {
		t.Fatalf("live session must survive: %v", err)
	}

	// Blob files for pruned rows are deleted; the missing one caused no
	// error either.
	for _, f := range []string{"blobs/dlv_old_delivered.json", "blobs/dlv_pro_old.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); !os.IsNotExist(err) {
			t.Fatalf("%s must be deleted", f)
		}
	}

	// A second sweep is a clean no-op.
	_, nDeliv, nErr, nSess, err = s.Sweep(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if nDeliv+nErr+nSess != 0 {
		t.Fatalf("second sweep counts %d %d %d", nDeliv, nErr, nSess)
	}
}

func TestRetentionSweeperRunStops(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st := memstore.New()
	blobs, err := blob.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &RetentionSweeper{
		Store:    st,
		Blobs:    blobs,
		Interval: time.Hour,
		Now:      func() time.Time { return now },
		Logf:     func(string, ...any) {},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop")
	}
}
