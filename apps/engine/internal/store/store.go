package store

import (
	"context"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
)

// Overview holds the dashboard overview numbers.
type Overview struct {
	ProjectsCount    int     `json:"projectsCount"`
	OpenIncidents    int     `json:"openIncidents"`
	PausedDeliveries int     `json:"pausedDeliveries"`
	CaughtToday      int     `json:"caughtToday"`
	HealthyRate      float64 `json:"healthyRate"`
}

// Store is the engine's data access boundary. The Postgres implementation is
// used in production; tests use the in-memory one.
type Store interface {
	// Projects
	CreateProject(ctx context.Context, p *model.Project) error
	ListProjects(ctx context.Context) ([]model.Project, error)
	GetProject(ctx context.Context, id string) (*model.Project, error)
	GetProjectByEndpointKey(ctx context.Context, key string) (*model.Project, error)
	GetProjectBySDKKey(ctx context.Context, key string) (*model.Project, error)

	// Deliveries
	CreateDelivery(ctx context.Context, d *model.Delivery) error
	GetDelivery(ctx context.Context, id string) (*model.Delivery, error)
	UpdateDelivery(ctx context.Context, d *model.Delivery) error
	ListDeliveries(ctx context.Context, f model.DeliveryFilter) ([]model.Delivery, error)
	// ClaimDueDeliveries leases due retrying deliveries so exactly one worker
	// handles them at a time.
	ClaimDueDeliveries(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]model.Delivery, error)
	// RecentDeliveredKeys returns payload key lists of recent healthy
	// deliveries for schema drift comparison.
	RecentDeliveredKeys(ctx context.Context, projectID string, limit int) ([][]string, error)

	// Incidents
	CreateIncident(ctx context.Context, i *model.Incident) error
	GetIncident(ctx context.Context, id string) (*model.Incident, error)
	ListIncidents(ctx context.Context, f model.IncidentFilter) ([]model.Incident, error)
	UpdateIncident(ctx context.Context, i *model.Incident) error
	// ExpireMutes flips muted incidents back to open after their mute window.
	ExpireMutes(ctx context.Context, now time.Time) (int, error)

	// Errors
	// RecordErrorEvent inserts the event and upserts its group, refreshing the
	// spike window counts.
	RecordErrorEvent(ctx context.Context, ev *model.ErrorEvent, window time.Duration) (*model.ErrorGroup, error)
	// ErrorEventByDelivery returns the newest error event linked to a delivery.
	ErrorEventByDelivery(ctx context.Context, deliveryID string) (*model.ErrorEvent, error)
	// LatestErrorEvent returns the newest error event in a group.
	LatestErrorEvent(ctx context.Context, groupID string) (*model.ErrorEvent, error)
	GetErrorGroup(ctx context.Context, id string) (*model.ErrorGroup, error)
	GetErrorGroupByFingerprint(ctx context.Context, projectID, fingerprint string) (*model.ErrorGroup, error)
	UpdateErrorGroup(ctx context.Context, g *model.ErrorGroup) error

	// Overview aggregates the dashboard numbers.
	Overview(ctx context.Context, now time.Time) (Overview, error)
}
