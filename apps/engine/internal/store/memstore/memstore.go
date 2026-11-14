// Package memstore is an in-memory Store used by tests and local smoke runs.
package memstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// Mem is a thread safe in-memory Store.
type Mem struct {
	mu         sync.Mutex
	projects   map[string]*model.Project
	deliveries map[string]*model.Delivery
	incidents  map[string]*model.Incident
	groups     map[string]*model.ErrorGroup
	events     map[string]*model.ErrorEvent
	nextGroup  int
}

// New returns an empty in-memory store.
func New() *Mem {
	return &Mem{
		projects:   map[string]*model.Project{},
		deliveries: map[string]*model.Delivery{},
		incidents:  map[string]*model.Incident{},
		groups:     map[string]*model.ErrorGroup{},
		events:     map[string]*model.ErrorEvent{},
	}
}

var _ store.Store = (*Mem)(nil)

func (m *Mem) CreateProject(_ context.Context, p *model.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.projects {
		if x.Slug == p.Slug || x.EndpointKey == p.EndpointKey || x.SDKKey == p.SDKKey {
			return fmt.Errorf("project already exists")
		}
	}
	cp := *p
	m.projects[p.ID] = &cp
	return nil
}

func (m *Mem) ListProjects(_ context.Context) ([]model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.Project, 0, len(m.projects))
	for _, p := range m.projects {
		out = append(out, *p)
	}
	return out, nil
}

func (m *Mem) GetProject(_ context.Context, id string) (*model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.projects[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, fmt.Errorf("project not found")
}

func (m *Mem) GetProjectByEndpointKey(_ context.Context, key string) (*model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.projects {
		if p.EndpointKey == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("project not found")
}

func (m *Mem) GetProjectBySDKKey(_ context.Context, key string) (*model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.projects {
		if p.SDKKey == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("project not found")
}

func (m *Mem) CreateDelivery(_ context.Context, d *model.Delivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *d
	cp.PayloadKeys = append([]string(nil), d.PayloadKeys...)
	cp.Headers = copyHeaders(d.Headers)
	m.deliveries[d.ID] = &cp
	return nil
}

func (m *Mem) GetDelivery(_ context.Context, id string) (*model.Delivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.deliveries[id]; ok {
		cp := *d
		cp.PayloadKeys = append([]string(nil), d.PayloadKeys...)
		cp.Headers = copyHeaders(d.Headers)
		return &cp, nil
	}
	return nil, fmt.Errorf("delivery not found")
}

func (m *Mem) UpdateDelivery(_ context.Context, d *model.Delivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deliveries[d.ID]; !ok {
		return fmt.Errorf("delivery not found")
	}
	cp := *d
	cp.PayloadKeys = append([]string(nil), d.PayloadKeys...)
	cp.Headers = copyHeaders(d.Headers)
	m.deliveries[d.ID] = &cp
	return nil
}

func (m *Mem) ListDeliveries(_ context.Context, f model.DeliveryFilter) ([]model.Delivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []model.Delivery
	for _, d := range m.deliveries {
		if f.ProjectID != "" && d.ProjectID != f.ProjectID {
			continue
		}
		if f.Status != "" && d.Status != f.Status {
			continue
		}
		if f.IncidentID != "" && d.LinkedIncidentID != f.IncidentID {
			continue
		}
		cp := *d
		cp.PayloadKeys = append([]string(nil), d.PayloadKeys...)
		cp.Headers = copyHeaders(d.Headers)
		out = append(out, cp)
	}
	sortDeliveries(out)
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (m *Mem) ClaimDueDeliveries(_ context.Context, now time.Time, lease time.Duration, limit int) ([]model.Delivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []model.Delivery
	for _, d := range m.deliveries {
		if len(out) >= limit {
			break
		}
		if d.Status != model.DeliveryRetrying || d.NextAttemptAt.After(now) {
			continue
		}
		d.NextAttemptAt = now.Add(lease)
		d.UpdatedAt = now
		cp := *d
		cp.PayloadKeys = append([]string(nil), d.PayloadKeys...)
		cp.Headers = copyHeaders(d.Headers)
		out = append(out, cp)
	}
	sortDeliveries(out)
	return out, nil
}

func (m *Mem) RecentDeliveredKeys(_ context.Context, projectID string, limit int) ([][]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []model.Delivery
	for _, d := range m.deliveries {
		if d.ProjectID == projectID && d.Status == model.DeliveryDelivered {
			all = append(all, *d)
		}
	}
	sortDeliveries(all)
	var out [][]string
	for _, d := range all {
		if len(out) >= limit {
			break
		}
		out = append(out, append([]string(nil), d.PayloadKeys...))
	}
	return out, nil
}

func (m *Mem) CreateIncident(_ context.Context, i *model.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *i
	m.incidents[i.ID] = &cp
	return nil
}

func (m *Mem) GetIncident(_ context.Context, id string) (*model.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i, ok := m.incidents[id]; ok {
		cp := *i
		return &cp, nil
	}
	return nil, fmt.Errorf("incident not found")
}

func (m *Mem) ListIncidents(_ context.Context, f model.IncidentFilter) ([]model.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []model.Incident
	for _, i := range m.incidents {
		if f.ProjectID != "" && i.ProjectID != f.ProjectID {
			continue
		}
		if f.AlertPhone != "" && i.LastAlertPhone != f.AlertPhone {
			continue
		}
		out = append(out, *i)
	}
	sortIncidents(out)
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (m *Mem) UpdateIncident(_ context.Context, i *model.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.incidents[i.ID]; !ok {
		return fmt.Errorf("incident not found")
	}
	cp := *i
	m.incidents[i.ID] = &cp
	return nil
}

func (m *Mem) ExpireMutes(_ context.Context, now time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, i := range m.incidents {
		if i.Status == model.IncidentMuted && i.MuteUntil != nil && !i.MuteUntil.After(now) {
			i.Status = model.IncidentOpen
			i.MuteUntil = nil
			n++
		}
	}
	return n, nil
}

func (m *Mem) RecordErrorEvent(_ context.Context, ev *model.ErrorEvent, window time.Duration) (*model.ErrorGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cpEv := *ev
	m.events[ev.ID] = &cpEv

	var g *model.ErrorGroup
	for _, x := range m.groups {
		if x.ProjectID == ev.ProjectID && x.Fingerprint == ev.Fingerprint {
			g = x
			break
		}
	}
	now := ev.CreatedAt
	if g == nil {
		m.nextGroup++
		g = &model.ErrorGroup{
			ID:              fmt.Sprintf("grp_%d", m.nextGroup),
			ProjectID:       ev.ProjectID,
			Fingerprint:     ev.Fingerprint,
			Title:           ev.Message,
			FilePath:        ev.FilePath,
			Line:            ev.Line,
			WindowStartedAt: now,
			FirstSeenAt:     now,
		}
		m.groups[g.ID] = g
	}
	g.EventCount++
	g.LastSeenAt = now
	if now.Sub(g.WindowStartedAt) > window {
		g.WindowCount = 1
		g.WindowStartedAt = now
	} else {
		g.WindowCount++
	}
	cpEv.GroupID = g.ID
	m.events[ev.ID] = &cpEv
	out := *g
	return &out, nil
}

func (m *Mem) ErrorEventByDelivery(_ context.Context, deliveryID string) (*model.ErrorEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newest *model.ErrorEvent
	for _, ev := range m.events {
		if ev.DeliveryID != deliveryID {
			continue
		}
		if newest == nil || ev.CreatedAt.After(newest.CreatedAt) {
			newest = ev
		}
	}
	if newest == nil {
		return nil, fmt.Errorf("error event not found")
	}
	cp := *newest
	return &cp, nil
}

func (m *Mem) LatestErrorEvent(_ context.Context, groupID string) (*model.ErrorEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newest *model.ErrorEvent
	for _, ev := range m.events {
		if ev.GroupID != groupID {
			continue
		}
		if newest == nil || ev.CreatedAt.After(newest.CreatedAt) {
			newest = ev
		}
	}
	if newest == nil {
		return nil, fmt.Errorf("error event not found")
	}
	cp := *newest
	return &cp, nil
}

func (m *Mem) GetErrorGroup(_ context.Context, id string) (*model.ErrorGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.groups[id]; ok {
		cp := *g
		return &cp, nil
	}
	return nil, fmt.Errorf("error group not found")
}

func (m *Mem) GetErrorGroupByFingerprint(_ context.Context, projectID, fingerprint string) (*model.ErrorGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range m.groups {
		if g.ProjectID == projectID && g.Fingerprint == fingerprint {
			cp := *g
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("error group not found")
}

func (m *Mem) UpdateErrorGroup(_ context.Context, g *model.ErrorGroup) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.groups[g.ID]; !ok {
		return fmt.Errorf("error group not found")
	}
	cp := *g
	m.groups[g.ID] = &cp
	return nil
}

func (m *Mem) Overview(_ context.Context, now time.Time) (store.Overview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o := store.Overview{HealthyRate: 100}
	seen := map[string]bool{}
	for _, p := range m.projects {
		if !seen[p.ID] {
			seen[p.ID] = true
			o.ProjectsCount++
		}
	}
	for _, i := range m.incidents {
		if i.Status == model.IncidentOpen {
			o.OpenIncidents++
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var finished, delivered int
	for _, d := range m.deliveries {
		if d.Status == model.DeliveryPaused {
			o.PausedDeliveries++
		}
		if !d.CreatedAt.Before(dayStart) {
			o.CaughtToday++
			switch d.Status {
			case model.DeliveryDelivered, model.DeliveryPaused, model.DeliveryFailed:
				finished++
				if d.Status == model.DeliveryDelivered {
					delivered++
				}
			}
		}
	}
	if finished > 0 {
		o.HealthyRate = float64(delivered*1000/finished) / 10
	} else {
		o.HealthyRate = 100
	}
	return o, nil
}

func copyHeaders(h map[string]string) map[string]string {
	if h == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}

func sortDeliveries(all []model.Delivery) {
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].CreatedAt.After(all[j-1].CreatedAt); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
}

func sortIncidents(all []model.Incident) {
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].LastSeenAt.After(all[j-1].LastSeenAt); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
}
