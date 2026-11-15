package memstore

import (
	"context"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

func TestWorkspaceUsageCounts(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	monthStart := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	st := New()

	_, wsp, err := st.UpsertGitHubUser(ctx, 21, "u", "U", "u@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	_, wsp2, err := st.UpsertGitHubUser(ctx, 22, "u2", "U2", "u2@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	proj := &model.Project{
		ID: "prj_u", Name: "u", Slug: "u", EndpointKey: "ep_u", SDKKey: "sdk_u",
		WorkspaceID: wsp.ID, Status: "active", CreatedAt: now,
	}
	other := &model.Project{
		ID: "prj_o", Name: "o", Slug: "o", EndpointKey: "ep_o", SDKKey: "sdk_o",
		WorkspaceID: wsp2.ID, Status: "active", CreatedAt: now,
	}
	for _, p := range []*model.Project{proj, other} {
		if err := st.CreateProject(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	// Two deliveries this month, one last month; one foreign delivery.
	rows := []*model.Delivery{
		{ID: "dlv_u_a", ProjectID: proj.ID, Status: model.DeliveryFailed, CreatedAt: now},
		{ID: "dlv_u_b", ProjectID: proj.ID, Status: model.DeliveryFailed, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "dlv_u_c", ProjectID: proj.ID, Status: model.DeliveryFailed, CreatedAt: now.Add(-45 * 24 * time.Hour)},
		{ID: "dlv_u_d", ProjectID: other.ID, Status: model.DeliveryFailed, CreatedAt: now},
	}
	for _, d := range rows {
		if err := st.CreateDelivery(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	// One error event this month in the workspace.
	if _, err := st.RecordErrorEvent(ctx, &model.ErrorEvent{
		ID: "err_u", ProjectID: proj.ID, Message: "m", Fingerprint: "fp",
		CreatedAt: now,
	}, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	projects, events, err := st.WorkspaceUsage(ctx, wsp.ID, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	if projects != 1 {
		t.Fatalf("projects %d", projects)
	}
	if events != 3 {
		t.Fatalf("monthly events %d want 3", events)
	}

	projects, events, err = st.WorkspaceUsage(ctx, wsp2.ID, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	if projects != 1 || events != 1 {
		t.Fatalf("other usage %d %d", projects, events)
	}
}

func TestUpgradeWorkspaceProIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st := New()
	_, wsp, err := st.UpsertGitHubUser(ctx, 22, "u2", "U2", "u2@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}

	expiry := now.Add(31 * 24 * time.Hour)
	if err := st.UpgradeWorkspacePro(ctx, wsp.ID, "ref_1", "cus_1", "sub_1", "NGN", 1500000, expiry); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetWorkspace(ctx, wsp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Plan != "pro" || got.PlanStatus != "active" || got.PlanExpiresAt == nil || !got.PlanExpiresAt.Equal(expiry) {
		t.Fatalf("after upgrade %+v", got)
	}

	// Same reference: replay is a no-op even with a different expiry.
	if err := st.UpgradeWorkspacePro(ctx, wsp.ID, "ref_1", "cus_1", "sub_1", "NGN", 1500000, now.Add(60*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _ = st.GetWorkspace(ctx, wsp.ID)
	if !got.PlanExpiresAt.Equal(expiry) {
		t.Fatalf("idempotent replay moved expiry: %v", got.PlanExpiresAt)
	}

	// Fresh reference: a later payment extends the window.
	if err := st.UpgradeWorkspacePro(ctx, wsp.ID, "ref_2", "cus_1", "sub_1", "NGN", 1500000, now.Add(60*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _ = st.GetWorkspace(ctx, wsp.ID)
	if !got.PlanExpiresAt.Equal(now.Add(60 * 24 * time.Hour)) {
		t.Fatalf("second payment must extend: %v", got.PlanExpiresAt)
	}
}

func TestCancelWorkspaceSubscription(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st := New()
	_, wsp, err := st.UpsertGitHubUser(ctx, 23, "u3", "U3", "u3@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}
	expiry := now.Add(31 * 24 * time.Hour)
	if err := st.UpgradeWorkspacePro(ctx, wsp.ID, "ref_c", "cus_c", "sub_c", "NGN", 1500000, expiry); err != nil {
		t.Fatal(err)
	}

	// Match by subscription code.
	if err := st.CancelWorkspaceSubscription(ctx, "sub_c"); err != nil {
		t.Fatal(err)
	}
	got, _ := st.GetWorkspace(ctx, wsp.ID)
	if got.PlanStatus != "cancelled" {
		t.Fatalf("status %q", got.PlanStatus)
	}
	if got.PlanExpiresAt == nil || !got.PlanExpiresAt.Equal(expiry) {
		t.Fatalf("expiry must stay intact: %v", got.PlanExpiresAt)
	}
	if err := st.CancelWorkspaceSubscription(ctx, "no_such_code"); err != store.ErrNotFound {
		t.Fatalf("unknown cancel err %v", err)
	}
	// Match by customer code also works on a fresh workspace.
	_, wsp2, err := st.UpsertGitHubUser(ctx, 24, "u4", "U4", "u4@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpgradeWorkspacePro(ctx, wsp2.ID, "ref_d", "cus_d", "sub_d", "NGN", 1500000, expiry); err != nil {
		t.Fatal(err)
	}
	if err := st.CancelWorkspaceSubscription(ctx, "cus_d"); err != nil {
		t.Fatal(err)
	}
	got2, _ := st.GetWorkspace(ctx, wsp2.ID)
	if got2.PlanStatus != "cancelled" {
		t.Fatalf("status %q", got2.PlanStatus)
	}
}

func TestPruneExpiredDataSemantics(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	st := New()

	_, wsp, err := st.UpsertGitHubUser(ctx, 25, "u5", "U5", "u5@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}
	proj := &model.Project{
		ID: "prj_p", Name: "p", Slug: "p", EndpointKey: "ep_p", SDKKey: "sdk_p",
		WorkspaceID: wsp.ID, Status: "active", CreatedAt: now,
	}
	if err := st.CreateProject(ctx, proj); err != nil {
		t.Fatal(err)
	}
	rows := []*model.Delivery{
		// Free retention is 3 days: 4 day old terminal rows go, paused and
		// retrying never go even when older.
		{ID: "dlv_a", ProjectID: proj.ID, Status: model.DeliveryDelivered, StorageKey: "k/a", CreatedAt: now.Add(-4 * day)},
		{ID: "dlv_b", ProjectID: proj.ID, Status: model.DeliveryFailed, StorageKey: "k/b", CreatedAt: now.Add(-9 * day)},
		{ID: "dlv_c", ProjectID: proj.ID, Status: model.DeliveryPaused, CreatedAt: now.Add(-9 * day)},
		{ID: "dlv_d", ProjectID: proj.ID, Status: model.DeliveryRetrying, AttemptCount: 1, CreatedAt: now.Add(-9 * day)},
		{ID: "dlv_e", ProjectID: proj.ID, Status: model.DeliveryDelivered, CreatedAt: now.Add(-2 * day)},
		{ID: "dlv_f", ProjectID: proj.ID, Status: model.DeliveryDelivered, StorageKey: "", CreatedAt: now.Add(-8 * day)},
	}
	for _, d := range rows {
		if err := st.CreateDelivery(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	groupOld, err := st.RecordErrorEvent(ctx, &model.ErrorEvent{
		ID: "err_p", ProjectID: proj.ID, Message: "m", Fingerprint: "fp_p",
		CreatedAt: now.Add(-4 * day),
	}, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	groupNew, err := st.RecordErrorEvent(ctx, &model.ErrorEvent{
		ID: "err_q", ProjectID: proj.ID, Message: "m", Fingerprint: "fp_q",
		CreatedAt: now.Add(-1 * day),
	}, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	keys, nDeliv, nErr, nSess, err := st.PruneExpiredData(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if nDeliv != 3 || nErr != 1 || nSess != 0 {
		t.Fatalf("counts %d %d %d", nDeliv, nErr, nSess)
	}
	// Only non-empty storage keys come back: k/a and k/b; dlv_f had none.
	got := map[string]bool{}
	for _, k := range keys {
		got[k] = true
	}
	if len(keys) != 2 || !got["k/a"] || !got["k/b"] {
		t.Fatalf("keys %v", keys)
	}
	for _, id := range []string{"dlv_c", "dlv_d", "dlv_e"} {
		if _, err := st.GetDelivery(ctx, id); err != nil {
			t.Fatalf("%s must survive: %v", id, err)
		}
	}
	for _, id := range []string{"dlv_a", "dlv_b", "dlv_f"} {
		if _, err := st.GetDelivery(ctx, id); err == nil {
			t.Fatalf("%s must be pruned", id)
		}
	}
	// Old error event gone, recent one kept.
	if _, err := st.LatestErrorEvent(ctx, groupOld.ID); err == nil {
		t.Fatal("old error event must be pruned")
	}
	if _, err := st.LatestErrorEvent(ctx, groupNew.ID); err != nil {
		t.Fatalf("recent error event must survive: %v", err)
	}
}
