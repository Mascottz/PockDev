// Package memstore is an in-memory Store used by tests and local smoke runs.
package memstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// Mem is a thread safe in-memory Store.
type Mem struct {
	mu           sync.Mutex
	projects     map[string]*model.Project
	deliveries   map[string]*model.Delivery
	incidents    map[string]*model.Incident
	groups       map[string]*model.ErrorGroup
	events       map[string]*model.ErrorEvent
	workspaces   map[string]*model.Workspace
	users        map[string]*model.User
	userByGitHub map[int64]string
	sessions     map[string]*model.Session
	nextGroup    int
}

// New returns an in-memory store seeded with the default workspace, matching
// the idempotent seed in schema.sql.
func New() *Mem {
	m := &Mem{
		projects:     map[string]*model.Project{},
		deliveries:   map[string]*model.Delivery{},
		incidents:    map[string]*model.Incident{},
		groups:       map[string]*model.ErrorGroup{},
		events:       map[string]*model.ErrorEvent{},
		workspaces:   map[string]*model.Workspace{},
		users:        map[string]*model.User{},
		userByGitHub: map[int64]string{},
		sessions:     map[string]*model.Session{},
	}
	m.workspaces[store.DefaultWorkspaceID] = &model.Workspace{
		ID:   store.DefaultWorkspaceID,
		Name: "Default Workspace",
		Slug: "default",
	}
	return m
}

var _ store.Store = (*Mem)(nil)

func workspaceOf(p *model.Project) string {
	if p == nil || p.WorkspaceID == "" {
		return store.DefaultWorkspaceID
	}
	return p.WorkspaceID
}

func (m *Mem) CreateProject(_ context.Context, p *model.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.WorkspaceID == "" {
		p.WorkspaceID = store.DefaultWorkspaceID
	}
	for _, x := range m.projects {
		if x.EndpointKey == p.EndpointKey || x.SDKKey == p.SDKKey {
			return fmt.Errorf("project already exists")
		}
		if x.Slug == p.Slug && workspaceOf(x) == p.WorkspaceID {
			return fmt.Errorf("project already exists")
		}
	}
	cp := *p
	m.projects[p.ID] = &cp
	return nil
}

func (m *Mem) ListProjects(_ context.Context, workspaceID string) ([]model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []model.Project{}
	for _, p := range m.projects {
		if workspaceOf(p) != workspaceID {
			continue
		}
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
	return nil, store.ErrNotFound
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
	return nil, store.ErrNotFound
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
	return nil, store.ErrNotFound
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
	return nil, store.ErrNotFound
}

func (m *Mem) UpdateDelivery(_ context.Context, d *model.Delivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deliveries[d.ID]; !ok {
		return store.ErrNotFound
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
		if f.WorkspaceID != "" {
			p, ok := m.projects[d.ProjectID]
			if !ok || workspaceOf(p) != f.WorkspaceID {
				continue
			}
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
	return nil, store.ErrNotFound
}

func (m *Mem) ListIncidents(_ context.Context, f model.IncidentFilter) ([]model.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []model.Incident
	for _, i := range m.incidents {
		if f.ProjectID != "" && i.ProjectID != f.ProjectID {
			continue
		}
		if f.WorkspaceID != "" {
			p, ok := m.projects[i.ProjectID]
			if !ok || workspaceOf(p) != f.WorkspaceID {
				continue
			}
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
		return store.ErrNotFound
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
		return nil, store.ErrNotFound
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
		return nil, store.ErrNotFound
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
	return nil, store.ErrNotFound
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
	return nil, store.ErrNotFound
}

func (m *Mem) UpdateErrorGroup(_ context.Context, g *model.ErrorGroup) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.groups[g.ID]; !ok {
		return store.ErrNotFound
	}
	cp := *g
	m.groups[g.ID] = &cp
	return nil
}

func (m *Mem) Overview(_ context.Context, workspaceID string, now time.Time) (store.Overview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o := store.Overview{HealthyRate: 100}
	for _, p := range m.projects {
		if workspaceOf(p) == workspaceID {
			o.ProjectsCount++
		}
	}
	for _, i := range m.incidents {
		if i.Status != model.IncidentOpen {
			continue
		}
		if p, ok := m.projects[i.ProjectID]; ok && workspaceOf(p) == workspaceID {
			o.OpenIncidents++
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var finished, delivered int
	for _, d := range m.deliveries {
		p, ok := m.projects[d.ProjectID]
		if !ok || workspaceOf(p) != workspaceID {
			continue
		}
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

func (m *Mem) UpsertGitHubUser(_ context.Context, ghID int64, login, name, email, avatarURL string, claimDefault bool) (*model.User, *model.Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if uid, ok := m.userByGitHub[ghID]; ok {
		u := m.users[uid]
		u.GitHubLogin = login
		u.Name = name
		u.Email = email
		u.AvatarURL = avatarURL
		cpU := *u
		w := m.workspaces[u.DefaultWorkspaceID]
		var cpW *model.Workspace
		if w != nil {
			wcp := *w
			cpW = &wcp
		}
		return &cpU, cpW, nil
	}

	userID := id.New("usr_", 16)
	var wsp *model.Workspace
	if claimDefault {
		def := m.workspaces[store.DefaultWorkspaceID]
		if def != nil && def.OwnerUserID == "" {
			def.OwnerUserID = userID
			if def.Name == "Default Workspace" {
				def.Name = login + " Workspace"
			}
			wsp = def
		}
	}
	if wsp == nil {
		wsp = m.insertWorkspaceLocked(userID, login+" Workspace", uniqueSlug(m, login))
	}
	u := &model.User{
		ID:                 userID,
		GitHubID:           ghID,
		GitHubLogin:        login,
		Name:               name,
		Email:              email,
		AvatarURL:          avatarURL,
		DefaultWorkspaceID: wsp.ID,
		CreatedAt:          time.Now(),
	}
	m.users[u.ID] = u
	m.userByGitHub[ghID] = u.ID
	cpU := *u
	cpW := *wsp
	return &cpU, &cpW, nil
}

func (m *Mem) insertWorkspaceLocked(ownerUserID, name, slug string) *model.Workspace {
	w := &model.Workspace{
		ID:          id.New("wsp_", 16),
		Name:        name,
		Slug:        slug,
		OwnerUserID: ownerUserID,
		CreatedAt:   time.Now(),
	}
	m.workspaces[w.ID] = w
	return w
}

func uniqueSlug(m *Mem, login string) string {
	base := store.Slugify(login)
	if base == "" {
		base = "user"
	}
	slug := base
	for i := 0; i < 5; i++ {
		taken := false
		for _, w := range m.workspaces {
			if w.Slug == slug {
				taken = true
				break
			}
		}
		if !taken {
			return slug
		}
		slug = base + "-" + id.Token(4)
	}
	return base + "-" + id.Token(8)
}

func (m *Mem) ListUserWorkspaces(_ context.Context, userID string) ([]model.WorkspaceSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []model.WorkspaceSummary{}
	for _, w := range m.workspaces {
		if w.OwnerUserID != userID {
			continue
		}
		out = append(out, model.WorkspaceSummary{ID: w.ID, Name: w.Name, Slug: w.Slug, Role: "owner"})
	}
	return out, nil
}

func (m *Mem) GetWorkspace(_ context.Context, id string) (*model.Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w, ok := m.workspaces[id]; ok {
		cp := *w
		return &cp, nil
	}
	return nil, store.ErrNotFound
}

func (m *Mem) CreateWorkspace(_ context.Context, userID, name, slug string) (*model.Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if slug == "" {
		slug = store.Slugify(name)
	}
	for _, w := range m.workspaces {
		if w.Slug == slug {
			return nil, fmt.Errorf("workspace with this slug already exists")
		}
	}
	w := m.insertWorkspaceLocked(userID, name, slug)
	cp := *w
	return &cp, nil
}

func (m *Mem) UserHasWorkspace(_ context.Context, userID, workspaceID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.workspaces[workspaceID]
	if !ok {
		return false, nil
	}
	return w.OwnerUserID == userID, nil
}

func (m *Mem) CreateSession(_ context.Context, sess *model.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *sess
	m.sessions[sess.TokenHash] = &cp
	return nil
}

func (m *Mem) GetSession(_ context.Context, tokenHash string, now time.Time) (*model.Session, *model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[tokenHash]
	if !ok || !s.ExpiresAt.After(now) {
		return nil, nil, store.ErrNotFound
	}
	u, ok := m.users[s.UserID]
	if !ok {
		return nil, nil, store.ErrNotFound
	}
	cpS := *s
	cpU := *u
	return &cpS, &cpU, nil
}

func (m *Mem) DeleteSession(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
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
