package memstore

import (
	"context"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
)

func TestOverviewNumbers(t *testing.T) {
	m := New()
	now := time.Now().UTC()
	_ = m.CreateProject(context.Background(), &model.Project{
		ID: "p1", Name: "a", Slug: "a", EndpointKey: "e1", SDKKey: "k1", Status: "active", CreatedAt: now,
	})
	_ = m.CreateProject(context.Background(), &model.Project{
		ID: "p2", Name: "b", Slug: "b", EndpointKey: "e2", SDKKey: "k2", Status: "active", CreatedAt: now,
	})
	_ = m.CreateIncident(context.Background(), &model.Incident{
		ID: "i1", ProjectID: "p1", Title: "t", Status: model.IncidentOpen, LastSeenAt: now, CreatedAt: now,
	})
	_ = m.CreateIncident(context.Background(), &model.Incident{
		ID: "i2", ProjectID: "p1", Title: "t", Status: model.IncidentMuted, LastSeenAt: now, CreatedAt: now,
	})
	for i, status := range []string{
		model.DeliveryDelivered, model.DeliveryDelivered,
		model.DeliveryPaused, model.DeliveryFailed,
		model.DeliveryRetrying,
	} {
		_ = m.CreateDelivery(context.Background(), &model.Delivery{
			ID: string(rune('a' + i)), ProjectID: "p1", Status: status, CreatedAt: now,
		})
	}
	// Yesterday's delivery must not count in today's numbers.
	_ = m.CreateDelivery(context.Background(), &model.Delivery{
		ID: "old", ProjectID: "p1", Status: model.DeliveryPaused,
		CreatedAt: now.Add(-25 * time.Hour),
	})

	o, err := m.Overview(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if o.ProjectsCount != 2 {
		t.Fatalf("projects %d", o.ProjectsCount)
	}
	if o.OpenIncidents != 1 {
		t.Fatalf("open incidents %d", o.OpenIncidents)
	}
	if o.PausedDeliveries != 2 {
		t.Fatalf("paused %d", o.PausedDeliveries)
	}
	if o.CaughtToday != 5 {
		t.Fatalf("caught today %d", o.CaughtToday)
	}
	if o.HealthyRate != 50 {
		t.Fatalf("healthy rate %v", o.HealthyRate)
	}
}

func TestRecordErrorEventWindowReset(t *testing.T) {
	m := New()
	start := time.Now().UTC()
	ev1 := &model.ErrorEvent{
		ID: "e1", ProjectID: "p1", Message: "boom", Fingerprint: "fp",
		CreatedAt: start,
	}
	g, err := m.RecordErrorEvent(context.Background(), ev1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if g.WindowCount != 1 || g.EventCount != 1 {
		t.Fatalf("group %+v", g)
	}

	ev2 := &model.ErrorEvent{
		ID: "e2", ProjectID: "p1", Message: "boom", Fingerprint: "fp",
		CreatedAt: start.Add(2 * time.Hour),
	}
	g, err = m.RecordErrorEvent(context.Background(), ev2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if g.WindowCount != 1 || g.EventCount != 2 {
		t.Fatalf("window should reset: %+v", g)
	}

	got, err := m.LatestErrorEvent(context.Background(), g.ID)
	if err != nil || got.ID != "e2" {
		t.Fatalf("latest %v %v", got, err)
	}
}

func TestDuplicateProjectRejected(t *testing.T) {
	m := New()
	p := &model.Project{ID: "p1", Name: "a", Slug: "a", EndpointKey: "e", SDKKey: "k", Status: "active"}
	if err := m.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if err := m.CreateProject(context.Background(), p); err == nil {
		t.Fatal("expected duplicate error")
	}
}
