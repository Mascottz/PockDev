package memstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

func TestUpsertGitHubUserAdminClaimsDefault(t *testing.T) {
	m := New()
	ctx := context.Background()

	u1, w1, err := m.UpsertGitHubUser(ctx, 1, "Mascottz", "Mascott", "m@example.com", "https://a/1.png", true)
	if err != nil {
		t.Fatal(err)
	}
	if w1.ID != store.DefaultWorkspaceID {
		t.Fatalf("admin must claim the default workspace, got %q", w1.ID)
	}
	if w1.Name != "Mascottz Workspace" || w1.Slug != "default" {
		t.Fatalf("claimed workspace %+v", w1)
	}
	if w1.OwnerUserID != u1.ID {
		t.Fatalf("owner %q want %q", w1.OwnerUserID, u1.ID)
	}

	mine, err := m.ListUserWorkspaces(ctx, u1.ID)
	if err != nil || len(mine) != 1 || mine[0].Role != "owner" || mine[0].ID != store.DefaultWorkspaceID {
		t.Fatalf("memberships %+v %v", mine, err)
	}

	// A second sign in with another login gets a fresh workspace.
	u2, w2, err := m.UpsertGitHubUser(ctx, 2, "alice", "Alice", "a@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if w2.ID == store.DefaultWorkspaceID {
		t.Fatal("second user must not steal the claimed default workspace")
	}
	if w2.Name != "alice Workspace" || w2.Slug != "alice" || w2.OwnerUserID != u2.ID {
		t.Fatalf("fresh workspace %+v", w2)
	}

	// claimDefault false also creates a fresh workspace.
	_, w3, err := m.UpsertGitHubUser(ctx, 3, "bob", "Bob", "b@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if w3.ID == store.DefaultWorkspaceID || w3.Slug != "bob" {
		t.Fatalf("fresh workspace %+v", w3)
	}

	// Returning user: same ids, profile refreshed.
	u1b, w1b, err := m.UpsertGitHubUser(ctx, 1, "Mascottz", "Renamed", "new@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if u1b.ID != u1.ID || w1b.ID != w1.ID {
		t.Fatalf("ids must not change: %+v %+v", u1b, w1b)
	}
	if u1b.Name != "Renamed" || u1b.Email != "new@example.com" {
		t.Fatalf("profile not updated: %+v", u1b)
	}
}

func TestSessionLifecycle(t *testing.T) {
	m := New()
	ctx := context.Background()
	u, w, err := m.UpsertGitHubUser(ctx, 9, "carol", "Carol", "c@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	hash := "deadbeef"
	if err := m.CreateSession(ctx, &model.Session{
		TokenHash: hash, UserID: u.ID, WorkspaceID: w.ID,
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	sess, got, err := m.GetSession(ctx, hash, now)
	if err != nil {
		t.Fatal(err)
	}
	if sess.WorkspaceID != w.ID || got.ID != u.ID {
		t.Fatalf("session %+v user %+v", sess, got)
	}

	// Expired sessions are reported as missing.
	if _, _, err := m.GetSession(ctx, hash, now.Add(2*time.Hour)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired: %v", err)
	}
	if _, _, err := m.GetSession(ctx, "unknown", now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}

	if err := m.DeleteSession(ctx, hash); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.GetSession(ctx, hash, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("deleted: %v", err)
	}
}

func TestWorkspaceMembership(t *testing.T) {
	m := New()
	ctx := context.Background()
	u, w, err := m.UpsertGitHubUser(ctx, 11, "dave", "Dave", "d@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := m.UserHasWorkspace(ctx, u.ID, w.ID)
	if err != nil || !ok {
		t.Fatalf("member: %v %v", ok, err)
	}
	ok, err = m.UserHasWorkspace(ctx, u.ID, store.DefaultWorkspaceID)
	if err != nil || ok {
		t.Fatalf("must not be member of default: %v %v", ok, err)
	}

	w2, err := m.CreateWorkspace(ctx, u.ID, "Extra", "extra")
	if err != nil {
		t.Fatal(err)
	}
	if w2.Slug != "extra" {
		t.Fatalf("workspace %+v", w2)
	}
	if ok, _ := m.UserHasWorkspace(ctx, u.ID, w2.ID); !ok {
		t.Fatal("owner must be member")
	}
	if _, err := m.CreateWorkspace(ctx, u.ID, "Extra2", "extra"); err == nil {
		t.Fatal("duplicate slug must fail")
	}
	list, err := m.ListUserWorkspaces(ctx, u.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("memberships %+v %v", list, err)
	}
}

func TestProjectWorkspaceScoping(t *testing.T) {
	m := New()
	ctx := context.Background()
	_, wA, err := m.UpsertGitHubUser(ctx, 21, "erin", "Erin", "e@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	_, wB, err := m.UpsertGitHubUser(ctx, 22, "frank", "Frank", "f@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	// Same slug in two workspaces is fine; key uniqueness stays global.
	for _, w := range []string{wA.ID, wB.ID} {
		wsid := w
		idx := "A"
		if wsid == wB.ID {
			idx = "B"
		}
		if err := m.CreateProject(ctx, &model.Project{
			ID: "p" + idx, WorkspaceID: wsid, Name: "Orders API", Slug: "orders-api",
			EndpointKey: "e" + idx, SDKKey: "k" + idx,
			Status: "active", CreatedAt: now,
		}); err != nil {
			t.Fatalf("same slug across workspaces: %v", err)
		}
	}
	if err := m.CreateProject(ctx, &model.Project{
		ID: "p3", WorkspaceID: wA.ID, Name: "x", Slug: "orders-api",
		EndpointKey: "e3", SDKKey: "k3", Status: "active", CreatedAt: now,
	}); err == nil {
		t.Fatal("same slug in one workspace must fail")
	}

	// Legacy rows land in the default workspace.
	if err := m.CreateProject(ctx, &model.Project{
		ID: "p4", Name: "legacy", Slug: "legacy", EndpointKey: "e4", SDKKey: "k4",
		Status: "active", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	listA, _ := m.ListProjects(ctx, wA.ID)
	listB, _ := m.ListProjects(ctx, wB.ID)
	listD, _ := m.ListProjects(ctx, store.DefaultWorkspaceID)
	if len(listA) != 1 || len(listB) != 1 || len(listD) != 1 {
		t.Fatalf("lists %d %d %d", len(listA), len(listB), len(listD))
	}
	if listA[0].WorkspaceID != wA.ID {
		t.Fatalf("project %+v", listA[0])
	}

	// Delivery and incident filters honor the workspace via the project.
	_ = m.CreateDelivery(ctx, &model.Delivery{ID: "dA", ProjectID: "pA", Status: model.DeliveryPaused, CreatedAt: now})
	_ = m.CreateDelivery(ctx, &model.Delivery{ID: "dB", ProjectID: "pB", Status: model.DeliveryPaused, CreatedAt: now})
	dels, _ := m.ListDeliveries(ctx, model.DeliveryFilter{WorkspaceID: wA.ID})
	if len(dels) != 1 || dels[0].ID != "dA" {
		t.Fatalf("deliveries %+v", dels)
	}
	_ = m.CreateIncident(ctx, &model.Incident{ID: "iB", ProjectID: "pB", Title: "t", Status: model.IncidentOpen, LastSeenAt: now, CreatedAt: now})
	incs, _ := m.ListIncidents(ctx, model.IncidentFilter{WorkspaceID: wB.ID})
	if len(incs) != 1 || incs[0].ID != "iB" {
		t.Fatalf("incidents %+v", incs)
	}

	o, _ := m.Overview(ctx, wA.ID, now)
	if o.ProjectsCount != 1 || o.PausedDeliveries != 1 || o.OpenIncidents != 0 {
		t.Fatalf("overview %+v", o)
	}
}
