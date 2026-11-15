package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
)

// ErrNotFound is returned when a row does not exist (or a session expired).
var ErrNotFound = errors.New("not found")

// Overview holds the dashboard overview numbers.
type Overview struct {
	ProjectsCount    int     `json:"projectsCount"`
	OpenIncidents    int     `json:"openIncidents"`
	PausedDeliveries int     `json:"pausedDeliveries"`
	CaughtToday      int     `json:"caughtToday"`
	HealthyRate      float64 `json:"healthyRate"`
}

// DefaultWorkspaceID is the seeded workspace that owns rows predating
// multi-tenancy.
const DefaultWorkspaceID = "wsp_default"

// Slugify turns a name or login into a URL safe slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// Store is the engine's data access boundary. The Postgres implementation is
// used in production; tests use the in-memory one.
type Store interface {
	// Projects
	CreateProject(ctx context.Context, p *model.Project) error
	// ListProjects returns only the projects owned by workspaceID.
	ListProjects(ctx context.Context, workspaceID string) ([]model.Project, error)
	GetProject(ctx context.Context, id string) (*model.Project, error)
	GetProjectByEndpointKey(ctx context.Context, key string) (*model.Project, error)
	GetProjectBySDKKey(ctx context.Context, key string) (*model.Project, error)

	// Deliveries
	CreateDelivery(ctx context.Context, d *model.Delivery) error
	GetDelivery(ctx context.Context, id string) (*model.Delivery, error)
	UpdateDelivery(ctx context.Context, d *model.Delivery) error
	// ListDeliveries filters deliveries. When f.WorkspaceID is set only
	// deliveries whose project lives in that workspace are returned.
	ListDeliveries(ctx context.Context, f model.DeliveryFilter) ([]model.Delivery, error)
	// ClaimDueDeliveries leases due retrying deliveries so exactly one worker
	// handles them at a time. It sweeps every workspace.
	ClaimDueDeliveries(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]model.Delivery, error)
	// RecentDeliveredKeys returns payload key lists of recent healthy
	// deliveries for schema drift comparison.
	RecentDeliveredKeys(ctx context.Context, projectID string, limit int) ([][]string, error)

	// Incidents
	CreateIncident(ctx context.Context, i *model.Incident) error
	GetIncident(ctx context.Context, id string) (*model.Incident, error)
	// ListIncidents filters incidents. When f.WorkspaceID is set only
	// incidents whose project lives in that workspace are returned.
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

	// Overview aggregates the dashboard numbers for one workspace only.
	Overview(ctx context.Context, workspaceID string, now time.Time) (Overview, error)

	// Auth and workspaces

	// UpsertGitHubUser signs in a GitHub account. Existing users get their
	// profile refreshed. New users get a workspace: when claimDefault is true
	// and the seeded default workspace has no owner they take it over,
	// otherwise a fresh workspace is created from their login.
	UpsertGitHubUser(ctx context.Context, ghID int64, login, name, email, avatarURL string, claimDefault bool) (*model.User, *model.Workspace, error)
	// ListUserWorkspaces returns every workspace the user belongs to.
	ListUserWorkspaces(ctx context.Context, userID string) ([]model.WorkspaceSummary, error)
	// GetWorkspace loads one workspace by id.
	GetWorkspace(ctx context.Context, id string) (*model.Workspace, error)
	// CreateWorkspace creates a workspace owned by userID and adds the owner
	// membership row.
	CreateWorkspace(ctx context.Context, userID, name, slug string) (*model.Workspace, error)
	// UserHasWorkspace reports whether the user is a member of the workspace.
	UserHasWorkspace(ctx context.Context, userID, workspaceID string) (bool, error)
	// CreateSession stores a session by its SHA-256 token hash.
	CreateSession(ctx context.Context, sess *model.Session) error
	// GetSession loads a session and its user, or ErrNotFound when the hash is
	// unknown or the session expired before now.
	GetSession(ctx context.Context, tokenHash string, now time.Time) (*model.Session, *model.User, error)
	// DeleteSession removes a session by token hash.
	DeleteSession(ctx context.Context, tokenHash string) error

	// Billing and retention

	// WorkspaceUsage counts the workspace projects and this month's events
	// (deliveries plus error events since monthStart).
	WorkspaceUsage(ctx context.Context, workspaceID string, monthStart time.Time) (projectsCount int, monthlyEvents int, err error)
	// UpgradeWorkspacePro flips the workspace to pro and records the payment
	// reference. Duplicate references are idempotent no-ops.
	UpgradeWorkspacePro(ctx context.Context, workspaceID, reference, customerCode, subscriptionCode, currency string, amountKobo int64, expiresAt time.Time) error
	// CancelWorkspaceSubscription marks the subscription cancelled for the
	// workspace owning the customer or subscription code, keeping the plan
	// active until plan_expires_at.
	CancelWorkspaceSubscription(ctx context.Context, customerOrSubCode string) error
	// PruneExpiredData deletes expired sessions, error events past their
	// workspace retention window, and terminal deliveries past retention. It
	// returns the storage keys of deleted deliveries so blobs can be removed.
	// Paused and retrying deliveries are never deleted.
	PruneExpiredData(ctx context.Context, now time.Time) (storageKeys []string, deletedDeliveries int, deletedErrors int, deletedSessions int, err error)
}
