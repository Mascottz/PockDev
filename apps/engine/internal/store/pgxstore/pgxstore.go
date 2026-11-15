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

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

func (s *Store) CreateProject(ctx context.Context, p *model.Project) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO projects (id, name, slug, endpoint_key, sdk_key, target_url, github_repo, whatsapp_phone, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		p.ID, p.Name, p.Slug, p.EndpointKey, p.SDKKey, p.TargetURL, p.GitHubRepo, p.WhatsAppPhone, p.Status, p.CreatedAt)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, slug, endpoint_key, sdk_key, target_url, github_repo, whatsapp_phone, status, created_at
		FROM projects ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Project{}
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.EndpointKey, &p.SDKKey, &p.TargetURL,
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
	var p model.Project
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, endpoint_key, sdk_key, target_url, github_repo, whatsapp_phone, status, created_at
		FROM projects `+where, arg).
		Scan(&p.ID, &p.Name, &p.Slug, &p.EndpointKey, &p.SDKKey, &p.TargetURL,
			&p.GitHubRepo, &p.WhatsAppPhone, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
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

func (s *Store) Overview(ctx context.Context, now time.Time) (store.Overview, error) {
	var o store.Overview
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM projects`).Scan(&o.ProjectsCount)
	if err != nil {
		return o, err
	}
	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE status = 'open'`).Scan(&o.OpenIncidents)
	if err != nil {
		return o, err
	}
	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM deliveries WHERE status = 'paused'`).Scan(&o.PausedDeliveries)
	if err != nil {
		return o, err
	}

	var delivered, finished int
	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'delivered'),
		       count(*) FILTER (WHERE status IN ('delivered', 'paused', 'failed'))
		FROM deliveries WHERE created_at >= $1`, dayStart).Scan(&delivered, &finished)
	if err != nil {
		return o, err
	}
	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM deliveries WHERE created_at >= $1`, dayStart).Scan(&o.CaughtToday)
	if err != nil {
		return o, err
	}

	o.HealthyRate = 100
	if finished > 0 {
		o.HealthyRate = float64(delivered*1000/finished) / 10
	}
	return o, nil
}
