// Package pgxstore is the production Postgres Store implementation.
package pgxstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// Store talks to Postgres through a pgx pool.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps a pgx pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

var _ store.Store = (*Store)(nil)

// ErrNotFound is returned when a row does not exist. It is the shared store
// sentinel so callers can use errors.Is across implementations.
var ErrNotFound = store.ErrNotFound

const projectCols = `id, workspace_id, name, slug, endpoint_key, sdk_key, target_url, github_repo, whatsapp_phone, status, created_at`

func scanProject(row pgx.Row) (*model.Project, error) {
	var p model.Project
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &p.EndpointKey, &p.SDKKey, &p.TargetURL,
		&p.GitHubRepo, &p.WhatsAppPhone, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func projectArgs(p *model.Project) []any {
	if p.WorkspaceID == "" {
		p.WorkspaceID = store.DefaultWorkspaceID
	}
	return []any{p.ID, p.WorkspaceID, p.Name, p.Slug, p.EndpointKey, p.SDKKey, p.TargetURL,
		p.GitHubRepo, p.WhatsAppPhone, p.Status, p.CreatedAt}
}

func (s *Store) CreateProject(ctx context.Context, p *model.Project) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO projects (`+projectCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, projectArgs(p)...)
	return err
}

func (s *Store) ListProjects(ctx context.Context, workspaceID string) ([]model.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+projectCols+`
		FROM projects WHERE workspace_id = $1 ORDER BY created_at ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Project{}
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &p.EndpointKey, &p.SDKKey, &p.TargetURL,
			&p.GitHubRepo, &p.WhatsAppPhone, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, id string) (*model.Project, error) {
	return s.oneProject(ctx, `WHERE id = $1`, id)
}

func (s *Store) GetProjectByEndpointKey(ctx context.Context, key string) (*model.Project, error) {
	return s.oneProject(ctx, `WHERE endpoint_key = $1`, key)
}

func (s *Store) GetProjectBySDKKey(ctx context.Context, key string) (*model.Project, error) {
	return s.oneProject(ctx, `WHERE sdk_key = $1`, key)
}

func (s *Store) oneProject(ctx context.Context, where, arg string) (*model.Project, error) {
	return scanProject(s.pool.QueryRow(ctx, `SELECT `+projectCols+` FROM projects `+where, arg))
}

const deliveryCols = `id, project_id, endpoint_key, status, http_status, attempt_count, schema_drift_note,
	payload_preview, payload_keys, headers, storage_key, linked_incident_id, last_error,
	next_attempt_at, created_at, updated_at, delivered_at`

func scanDelivery(row pgx.Row) (*model.Delivery, error) {
	var d model.Delivery
	var keysJSON, headersJSON []byte
	err := row.Scan(&d.ID, &d.ProjectID, &d.EndpointKey, &d.Status, &d.HTTPStatus, &d.AttemptCount,
		&d.SchemaDriftNote, &d.PayloadPreview, &keysJSON, &headersJSON, &d.StorageKey,
		&d.LinkedIncidentID, &d.LastError, &d.NextAttemptAt, &d.CreatedAt, &d.UpdatedAt, &d.DeliveredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(keysJSON) > 0 {
		_ = json.Unmarshal(keysJSON, &d.PayloadKeys)
	}
	if len(headersJSON) > 0 {
		_ = json.Unmarshal(headersJSON, &d.Headers)
	}
	if d.PayloadKeys == nil {
		d.PayloadKeys = []string{}
	}
	if d.Headers == nil {
		d.Headers = map[string]string{}
	}
	return &d, nil
}

func deliveryArgs(d *model.Delivery) ([]any, error) {
	keysJSON, err := json.Marshal(d.PayloadKeys)
	if err != nil {
		return nil, err
	}
	if d.Headers == nil {
		d.Headers = map[string]string{}
	}
	headersJSON, err := json.Marshal(d.Headers)
	if err != nil {
		return nil, err
	}
	return []any{d.ID, d.ProjectID, d.EndpointKey, d.Status, d.HTTPStatus, d.AttemptCount,
		d.SchemaDriftNote, d.PayloadPreview, keysJSON, headersJSON, d.StorageKey,
		d.LinkedIncidentID, d.LastError, d.NextAttemptAt, d.CreatedAt, d.UpdatedAt, d.DeliveredAt}, nil
}

func (s *Store) CreateDelivery(ctx context.Context, d *model.Delivery) error {
	args, err := deliveryArgs(d)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO deliveries (`+deliveryCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`, args...)
	return err
}

func (s *Store) GetDelivery(ctx context.Context, id string) (*model.Delivery, error) {
	return scanDelivery(s.pool.QueryRow(ctx, `SELECT `+deliveryCols+` FROM deliveries WHERE id = $1`, id))
}

func (s *Store) UpdateDelivery(ctx context.Context, d *model.Delivery) error {
	args, err := deliveryArgs(d)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE deliveries SET
			status = $2, http_status = $3, attempt_count = $4, schema_drift_note = $5,
			payload_preview = $6, payload_keys = $7, headers = $8, storage_key = $9,
			linked_incident_id = $10, last_error = $11, next_attempt_at = $12,
			updated_at = $13, delivered_at = $14
		WHERE id = $1`,
		args[0], args[3], args[4], args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[13], args[15], args[16])
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListDeliveries(ctx context.Context, f model.DeliveryFilter) ([]model.Delivery, error) {
	query := `SELECT ` + deliveryCols + ` FROM deliveries WHERE 1=1`
	args := []any{}
	if f.ProjectID != "" {
		args = append(args, f.ProjectID)
		query += fmt.Sprintf(` AND project_id = $%d`, len(args))
	}
	if f.WorkspaceID != "" {
		args = append(args, f.WorkspaceID)
		query += fmt.Sprintf(` AND project_id IN (SELECT id FROM projects WHERE workspace_id = $%d)`, len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		query += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	if f.IncidentID != "" {
		args = append(args, f.IncidentID)
		query += fmt.Sprintf(` AND linked_incident_id = $%d`, len(args))
	}
	query += ` ORDER BY created_at DESC`
	if f.Limit > 0 {
		args = append(args, f.Limit)
		query += fmt.Sprintf(` LIMIT $%d`, len(args))
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Delivery{}
	for rows.Next() {
		var d model.Delivery
		var keysJSON, headersJSON []byte
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.EndpointKey, &d.Status, &d.HTTPStatus, &d.AttemptCount,
			&d.SchemaDriftNote, &d.PayloadPreview, &keysJSON, &headersJSON, &d.StorageKey,
			&d.LinkedIncidentID, &d.LastError, &d.NextAttemptAt, &d.CreatedAt, &d.UpdatedAt, &d.DeliveredAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(keysJSON, &d.PayloadKeys)
		_ = json.Unmarshal(headersJSON, &d.Headers)
		if d.PayloadKeys == nil {
			d.PayloadKeys = []string{}
		}
		if d.Headers == nil {
			d.Headers = map[string]string{}
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) ClaimDueDeliveries(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]model.Delivery, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE deliveries d SET next_attempt_at = $2, updated_at = $1
		FROM (
			SELECT id FROM deliveries
			WHERE status = 'retrying' AND next_attempt_at <= $1
			ORDER BY next_attempt_at
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		) c
		WHERE d.id = c.id
		RETURNING `+prefixedDeliveryCols("d"), now, now.Add(lease), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Delivery{}
	for rows.Next() {
		var d model.Delivery
		var keysJSON, headersJSON []byte
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.EndpointKey, &d.Status, &d.HTTPStatus, &d.AttemptCount,
			&d.SchemaDriftNote, &d.PayloadPreview, &keysJSON, &headersJSON, &d.StorageKey,
			&d.LinkedIncidentID, &d.LastError, &d.NextAttemptAt, &d.CreatedAt, &d.UpdatedAt, &d.DeliveredAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(keysJSON, &d.PayloadKeys)
		_ = json.Unmarshal(headersJSON, &d.Headers)
		out = append(out, d)
	}
	return out, rows.Err()
}

func prefixedDeliveryCols(alias string) string {
	cols := []string{"id", "project_id", "endpoint_key", "status", "http_status", "attempt_count",
		"schema_drift_note", "payload_preview", "payload_keys", "headers", "storage_key",
		"linked_incident_id", "last_error", "next_attempt_at", "created_at", "updated_at", "delivered_at"}
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += alias + "." + c
	}
	return out
}

func (s *Store) RecentDeliveredKeys(ctx context.Context, projectID string, limit int) ([][]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT payload_keys FROM deliveries
		WHERE project_id = $1 AND status = 'delivered'
		ORDER BY created_at DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		keys := []string{}
		_ = json.Unmarshal(raw, &keys)
		out = append(out, keys)
	}
	return out, rows.Err()
}

const incidentCols = `id, project_id, title, plain_summary, file_path, commit_sha, status,
	paused_count, pr_url, origin, origin_ref, mute_until, last_alert_phone, last_alert_at,
	last_seen_at, created_at`

func scanIncident(row pgx.Row) (*model.Incident, error) {
	var i model.Incident
	err := row.Scan(&i.ID, &i.ProjectID, &i.Title, &i.PlainSummary, &i.FilePath, &i.CommitSHA,
		&i.Status, &i.PausedCount, &i.PRURL, &i.Origin, &i.OriginRef, &i.MuteUntil,
		&i.LastAlertPhone, &i.LastAlertAt, &i.LastSeenAt, &i.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (s *Store) CreateIncident(ctx context.Context, i *model.Incident) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO incidents (`+incidentCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		i.ID, i.ProjectID, i.Title, i.PlainSummary, i.FilePath, i.CommitSHA, i.Status,
		i.PausedCount, i.PRURL, i.Origin, i.OriginRef, i.MuteUntil, i.LastAlertPhone,
		i.LastAlertAt, i.LastSeenAt, i.CreatedAt)
	return err
}

func (s *Store) GetIncident(ctx context.Context, id string) (*model.Incident, error) {
	return scanIncident(s.pool.QueryRow(ctx, `SELECT `+incidentCols+` FROM incidents WHERE id = $1`, id))
}

func (s *Store) ListIncidents(ctx context.Context, f model.IncidentFilter) ([]model.Incident, error) {
	query := `SELECT ` + incidentCols + ` FROM incidents WHERE 1=1`
	args := []any{}
	if f.ProjectID != "" {
		args = append(args, f.ProjectID)
		query += fmt.Sprintf(` AND project_id = $%d`, len(args))
	}
	if f.WorkspaceID != "" {
		args = append(args, f.WorkspaceID)
		query += fmt.Sprintf(` AND project_id IN (SELECT id FROM projects WHERE workspace_id = $%d)`, len(args))
	}
	if f.AlertPhone != "" {
		args = append(args, f.AlertPhone)
		query += fmt.Sprintf(` AND last_alert_phone = $%d`, len(args))
	}
	query += ` ORDER BY last_seen_at DESC`
	if f.Limit > 0 {
		args = append(args, f.Limit)
		query += fmt.Sprintf(` LIMIT $%d`, len(args))
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Incident{}
	for rows.Next() {
		var i model.Incident
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Title, &i.PlainSummary, &i.FilePath, &i.CommitSHA,
			&i.Status, &i.PausedCount, &i.PRURL, &i.Origin, &i.OriginRef, &i.MuteUntil,
			&i.LastAlertPhone, &i.LastAlertAt, &i.LastSeenAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) UpdateIncident(ctx context.Context, i *model.Incident) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE incidents SET
			title = $2, plain_summary = $3, file_path = $4, commit_sha = $5, status = $6,
			paused_count = $7, pr_url = $8, origin = $9, origin_ref = $10, mute_until = $11,
			last_alert_phone = $12, last_alert_at = $13, last_seen_at = $14
		WHERE id = $1`,
		i.ID, i.Title, i.PlainSummary, i.FilePath, i.CommitSHA, i.Status,
		i.PausedCount, i.PRURL, i.Origin, i.OriginRef, i.MuteUntil,
		i.LastAlertPhone, i.LastAlertAt, i.LastSeenAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ExpireMutes(ctx context.Context, now time.Time) (int, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE incidents SET status = 'open', mute_until = NULL
		WHERE status = 'muted' AND mute_until IS NOT NULL AND mute_until <= $1`, now)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *Store) RecordErrorEvent(ctx context.Context, ev *model.ErrorEvent, window time.Duration) (*model.ErrorGroup, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var g model.ErrorGroup
	err = tx.QueryRow(ctx, `
		SELECT id, project_id, fingerprint, title, file_path, line, event_count, window_count,
			window_started_at, incident_id, first_seen_at, last_seen_at
		FROM error_groups WHERE project_id = $1 AND fingerprint = $2
		FOR UPDATE`, ev.ProjectID, ev.Fingerprint).
		Scan(&g.ID, &g.ProjectID, &g.Fingerprint, &g.Title, &g.FilePath, &g.Line, &g.EventCount,
			&g.WindowCount, &g.WindowStartedAt, &g.IncidentID, &g.FirstSeenAt, &g.LastSeenAt)

	now := ev.CreatedAt
	if errors.Is(err, pgx.ErrNoRows) {
		g = model.ErrorGroup{
			ID:              ev.ID + "_g",
			ProjectID:       ev.ProjectID,
			Fingerprint:     ev.Fingerprint,
			Title:           ev.Message,
			FilePath:        ev.FilePath,
			Line:            ev.Line,
			WindowStartedAt: now,
			FirstSeenAt:     now,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO error_groups (id, project_id, fingerprint, title, file_path, line,
				event_count, window_count, window_started_at, incident_id, first_seen_at, last_seen_at)
			VALUES ($1, $2, $3, $4, $5, $6, 1, 1, $7, '', $8, $8)`,
			g.ID, g.ProjectID, g.Fingerprint, g.Title, g.FilePath, g.Line,
			g.WindowStartedAt, now)
		if err != nil {
			return nil, err
		}
		g.EventCount = 1
		g.WindowCount = 1
	} else if err != nil {
		return nil, err
	} else {
		g.EventCount++
		g.LastSeenAt = now
		if now.Sub(g.WindowStartedAt) > window {
			g.WindowCount = 1
			g.WindowStartedAt = now
		} else {
			g.WindowCount++
		}
		_, err = tx.Exec(ctx, `
			UPDATE error_groups SET event_count = $2, window_count = $3, window_started_at = $4, last_seen_at = $5
			WHERE id = $1`, g.ID, g.EventCount, g.WindowCount, g.WindowStartedAt, now)
		if err != nil {
			return nil, err
		}
	}

	ev.GroupID = g.ID
	_, err = tx.Exec(ctx, `
		INSERT INTO error_events (id, project_id, group_id, message, stack, file_path, line, delivery_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		ev.ID, ev.ProjectID, g.ID, ev.Message, ev.Stack, ev.FilePath, ev.Line, ev.DeliveryID, ev.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	g.LastSeenAt = now
	return &g, nil
}

func (s *Store) GetErrorGroup(ctx context.Context, id string) (*model.ErrorGroup, error) {
	var g model.ErrorGroup
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, fingerprint, title, file_path, line, event_count, window_count,
			window_started_at, incident_id, first_seen_at, last_seen_at
		FROM error_groups WHERE id = $1`, id).
		Scan(&g.ID, &g.ProjectID, &g.Fingerprint, &g.Title, &g.FilePath, &g.Line, &g.EventCount,
			&g.WindowCount, &g.WindowStartedAt, &g.IncidentID, &g.FirstSeenAt, &g.LastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) GetErrorGroupByFingerprint(ctx context.Context, projectID, fingerprint string) (*model.ErrorGroup, error) {
	var g model.ErrorGroup
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, fingerprint, title, file_path, line, event_count, window_count,
			window_started_at, incident_id, first_seen_at, last_seen_at
		FROM error_groups WHERE project_id = $1 AND fingerprint = $2`, projectID, fingerprint).
		Scan(&g.ID, &g.ProjectID, &g.Fingerprint, &g.Title, &g.FilePath, &g.Line, &g.EventCount,
			&g.WindowCount, &g.WindowStartedAt, &g.IncidentID, &g.FirstSeenAt, &g.LastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) UpdateErrorGroup(ctx context.Context, g *model.ErrorGroup) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE error_groups SET title = $2, file_path = $3, line = $4, event_count = $5,
			window_count = $6, window_started_at = $7, incident_id = $8, last_seen_at = $9
		WHERE id = $1`,
		g.ID, g.Title, g.FilePath, g.Line, g.EventCount, g.WindowCount,
		g.WindowStartedAt, g.IncidentID, g.LastSeenAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ErrorEventByDelivery(ctx context.Context, deliveryID string) (*model.ErrorEvent, error) {
	return s.oneErrorEvent(ctx, `WHERE delivery_id = $1 ORDER BY created_at DESC LIMIT 1`, deliveryID)
}

func (s *Store) LatestErrorEvent(ctx context.Context, groupID string) (*model.ErrorEvent, error) {
	return s.oneErrorEvent(ctx, `WHERE group_id = $1 ORDER BY created_at DESC LIMIT 1`, groupID)
}

func (s *Store) oneErrorEvent(ctx context.Context, tail, arg string) (*model.ErrorEvent, error) {
	var ev model.ErrorEvent
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, group_id, message, stack, file_path, line, delivery_id, created_at
		FROM error_events `+tail, arg).
		Scan(&ev.ID, &ev.ProjectID, &ev.GroupID, &ev.Message, &ev.Stack, &ev.FilePath,
			&ev.Line, &ev.DeliveryID, &ev.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (s *Store) Overview(ctx context.Context, workspaceID string, now time.Time) (store.Overview, error) {
	var o store.Overview
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM projects WHERE workspace_id = $1`, workspaceID).Scan(&o.ProjectsCount)
	if err != nil {
		return o, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FROM incidents
		WHERE status = 'open' AND project_id IN (SELECT id FROM projects WHERE workspace_id = $1)`,
		workspaceID).Scan(&o.OpenIncidents)
	if err != nil {
		return o, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FROM deliveries
		WHERE status = 'paused' AND project_id IN (SELECT id FROM projects WHERE workspace_id = $1)`,
		workspaceID).Scan(&o.PausedDeliveries)
	if err != nil {
		return o, err
	}

	var delivered, finished int
	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'delivered'),
		       count(*) FILTER (WHERE status IN ('delivered', 'paused', 'failed'))
		FROM deliveries
		WHERE created_at >= $1 AND project_id IN (SELECT id FROM projects WHERE workspace_id = $2)`,
		dayStart, workspaceID).Scan(&delivered, &finished)
	if err != nil {
		return o, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FROM deliveries
		WHERE created_at >= $1 AND project_id IN (SELECT id FROM projects WHERE workspace_id = $2)`,
		dayStart, workspaceID).Scan(&o.CaughtToday)
	if err != nil {
		return o, err
	}

	o.HealthyRate = 100
	if finished > 0 {
		o.HealthyRate = float64(delivered*1000/finished) / 10
	}
	return o, nil
}

const workspaceCols = `id, name, slug, owner_user_id, created_at, plan, plan_status, plan_expires_at, paystack_customer_code, paystack_subscription_code`

func scanWorkspace(row pgx.Row) (*model.Workspace, error) {
	var w model.Workspace
	err := row.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerUserID, &w.CreatedAt,
		&w.Plan, &w.PlanStatus, &w.PlanExpiresAt, &w.PaystackCustomerCode, &w.PaystackSubscriptionCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

const userCols = `id, github_id, github_login, name, email, avatar_url, default_workspace_id, created_at`

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.GitHubID, &u.GitHubLogin, &u.Name, &u.Email, &u.AvatarURL,
		&u.DefaultWorkspaceID, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpsertGitHubUser(ctx context.Context, ghID int64, login, name, email, avatarURL string, claimDefault bool) (*model.User, *model.Workspace, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE github_id = $1`, ghID))
	if errors.Is(err, ErrNotFound) {
		userID := id.New("usr_", 16)
		var wsp *model.Workspace
		if claimDefault {
			wsp, err = scanWorkspace(tx.QueryRow(ctx, `
				UPDATE workspaces SET owner_user_id = $2,
					name = CASE WHEN name = 'Default Workspace' THEN $3 ELSE name END
				WHERE id = $1 AND owner_user_id = ''
				RETURNING `+workspaceCols, store.DefaultWorkspaceID, userID, login+" Workspace"))
			if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, nil, err
			}
		}
		if wsp == nil {
			wsp, err = insertWorkspaceTx(ctx, tx, userID, login+" Workspace", uniqueWorkspaceSlug(ctx, tx, login))
			if err != nil {
				return nil, nil, err
			}
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO users (`+userCols+`)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			userID, ghID, login, name, email, avatarURL, wsp.ID, time.Now())
		if err != nil {
			return nil, nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role, created_at)
			VALUES ($1, $2, 'owner', $3)`, wsp.ID, userID, time.Now())
		if err != nil {
			return nil, nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, nil, err
		}
		u = &model.User{
			ID: userID, GitHubID: ghID, GitHubLogin: login, Name: name, Email: email,
			AvatarURL: avatarURL, DefaultWorkspaceID: wsp.ID,
		}
		return u, wsp, nil
	}
	if err != nil {
		return nil, nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE users SET github_login = $2, name = $3, email = $4, avatar_url = $5
		WHERE id = $1`, u.ID, login, name, email, avatarURL)
	if err != nil {
		return nil, nil, err
	}
	u.GitHubLogin = login
	u.Name = name
	u.Email = email
	u.AvatarURL = avatarURL
	wsp, err := scanWorkspace(tx.QueryRow(ctx, `SELECT `+workspaceCols+` FROM workspaces WHERE id = $1`, u.DefaultWorkspaceID))
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return u, wsp, nil
}

func insertWorkspaceTx(ctx context.Context, tx pgx.Tx, ownerUserID, name, slug string) (*model.Workspace, error) {
	w := &model.Workspace{
		ID:          id.New("wsp_", 16),
		Name:        name,
		Slug:        slug,
		OwnerUserID: ownerUserID,
		CreatedAt:   time.Now(),
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO workspaces (id, name, slug, owner_user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)`, w.ID, w.Name, w.Slug, w.OwnerUserID, w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// uniqueWorkspaceSlug finds a free slug derived from the GitHub login.
func uniqueWorkspaceSlug(ctx context.Context, tx pgx.Tx, login string) string {
	base := store.Slugify(login)
	if base == "" {
		base = "user"
	}
	slug := base
	for i := 0; i < 5; i++ {
		var taken int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM workspaces WHERE slug = $1`, slug).Scan(&taken)
		if err == nil && taken == 0 {
			return slug
		}
		slug = base + "-" + id.Token(4)
	}
	return base + "-" + id.Token(8)
}

func (s *Store) ListUserWorkspaces(ctx context.Context, userID string) ([]model.WorkspaceSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, m.role
		FROM workspace_members m
		JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id = $1
		ORDER BY w.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.WorkspaceSummary{}
	for rows.Next() {
		var w model.WorkspaceSummary
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.Role); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspace(ctx context.Context, id string) (*model.Workspace, error) {
	return scanWorkspace(s.pool.QueryRow(ctx, `SELECT `+workspaceCols+` FROM workspaces WHERE id = $1`, id))
}

func (s *Store) CreateWorkspace(ctx context.Context, userID, name, slug string) (*model.Workspace, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if slug == "" {
		slug = store.Slugify(name)
	}
	w, err := insertWorkspaceTx(ctx, tx, userID, name, slug)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES ($1, $2, 'owner', $3)`, w.ID, userID, time.Now())
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Store) UserHasWorkspace(ctx context.Context, userID, workspaceID string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM workspace_members
		WHERE user_id = $1 AND workspace_id = $2`, userID, workspaceID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) CreateSession(ctx context.Context, sess *model.Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, workspace_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		sess.TokenHash, sess.UserID, sess.WorkspaceID, sess.ExpiresAt, sess.CreatedAt)
	return err
}

func (s *Store) GetSession(ctx context.Context, tokenHash string, now time.Time) (*model.Session, *model.User, error) {
	var sess model.Session
	var u model.User
	err := s.pool.QueryRow(ctx, `
		SELECT s.token_hash, s.user_id, s.workspace_id, s.expires_at, s.created_at,
			u.id, u.github_id, u.github_login, u.name, u.email, u.avatar_url, u.default_workspace_id, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now).
		Scan(&sess.TokenHash, &sess.UserID, &sess.WorkspaceID, &sess.ExpiresAt, &sess.CreatedAt,
			&u.ID, &u.GitHubID, &u.GitHubLogin, &u.Name, &u.Email, &u.AvatarURL, &u.DefaultWorkspaceID, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &sess, &u, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) WorkspaceUsage(ctx context.Context, workspaceID string, monthStart time.Time) (int, int, error) {
	var projects int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM projects WHERE workspace_id = $1`, workspaceID).Scan(&projects); err != nil {
		return 0, 0, err
	}
	var events int
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM deliveries WHERE created_at >= $1
				AND project_id IN (SELECT id FROM projects WHERE workspace_id = $2)) +
			(SELECT count(*) FROM error_events WHERE created_at >= $1
				AND project_id IN (SELECT id FROM projects WHERE workspace_id = $2))`,
		monthStart, workspaceID).Scan(&events)
	if err != nil {
		return 0, 0, err
	}
	return projects, events, nil
}

func (s *Store) UpgradeWorkspacePro(ctx context.Context, workspaceID, reference, customerCode, subscriptionCode, currency string, amountKobo int64, expiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		INSERT INTO billing_events (reference, workspace_id, amount_kobo, currency, status, created_at)
		VALUES ($1, $2, $3, $4, 'success', $5)
		ON CONFLICT (reference) DO NOTHING`,
		reference, workspaceID, amountKobo, currency, time.Now())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Reference already recorded: the upgrade is idempotent.
		return nil
	}
	tag, err = tx.Exec(ctx, `
		UPDATE workspaces SET
			plan = 'pro', plan_status = 'active', plan_expires_at = $2,
			paystack_customer_code = CASE WHEN $3 <> '' THEN $3 ELSE paystack_customer_code END,
			paystack_subscription_code = CASE WHEN $4 <> '' THEN $4 ELSE paystack_subscription_code END
		WHERE id = $1`,
		workspaceID, expiresAt, customerCode, subscriptionCode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Store) CancelWorkspaceSubscription(ctx context.Context, customerOrSubCode string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE workspaces SET plan_status = 'cancelled'
		WHERE paystack_customer_code = $1 OR paystack_subscription_code = $1`, customerOrSubCode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const retentionCutoffSQL = `$1::timestamptz - (CASE WHEN w.plan = 'pro' AND (w.plan_expires_at IS NULL OR w.plan_expires_at > $1) THEN 30 ELSE 3 END) * interval '1 day'`

func (s *Store) PruneExpiredData(ctx context.Context, now time.Time) ([]string, int, int, int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	deletedSessions := int(tag.RowsAffected())

	tag, err = tx.Exec(ctx, `
		DELETE FROM error_events e
		USING projects p, workspaces w
		WHERE p.id = e.project_id AND w.id = p.workspace_id
			AND e.created_at < `+retentionCutoffSQL, now)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	deletedErrors := int(tag.RowsAffected())

	rows, err := tx.Query(ctx, `
		WITH gone AS (
			DELETE FROM deliveries d
			USING projects p, workspaces w
			WHERE p.id = d.project_id AND w.id = p.workspace_id
				AND d.status IN ('delivered', 'failed')
				AND d.created_at < `+retentionCutoffSQL+`
			RETURNING d.storage_key
		)
		SELECT storage_key FROM gone`, now)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer rows.Close()
	storageKeys := []string{}
	deletedDeliveries := 0
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, 0, 0, 0, err
		}
		deletedDeliveries++
		if key != "" {
			storageKeys = append(storageKeys, key)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, 0, 0, err
	}
	return storageKeys, deletedDeliveries, deletedErrors, deletedSessions, nil
}
