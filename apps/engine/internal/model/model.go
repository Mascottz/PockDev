package model

import "time"

// Project is one app or service being watched by PockDev.
type Project struct {
	ID            string    `json:"id"`
	WorkspaceID   string    `json:"workspaceId"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	EndpointKey   string    `json:"endpointKey"`
	SDKKey        string    `json:"sdkKey"`
	TargetURL     string    `json:"targetUrl"`
	GitHubRepo    string    `json:"githubRepo"`
	WhatsAppPhone string    `json:"whatsappPhone"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Delivery status values shared with the dashboard contract.
const (
	DeliveryDelivered = "delivered"
	DeliveryRetrying  = "retrying"
	DeliveryPaused    = "paused"
	DeliveryFailed    = "failed"
)

// Delivery is one caught webhook on its way to the project target URL.
type Delivery struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"projectId"`
	EndpointKey      string            `json:"-"`
	Status           string            `json:"status"`
	HTTPStatus       int               `json:"httpStatus"`
	AttemptCount     int               `json:"attemptCount"`
	SchemaDriftNote  string            `json:"schemaDriftNote"`
	PayloadPreview   string            `json:"payloadPreview"`
	PayloadKeys      []string          `json:"-"`
	Headers          map[string]string `json:"-"`
	StorageKey       string            `json:"-"`
	LinkedIncidentID string            `json:"linkedIncidentId"`
	LastError        string            `json:"-"`
	NextAttemptAt    time.Time         `json:"-"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"-"`
	DeliveredAt      *time.Time        `json:"-"`
}

// Incident status values shared with the dashboard contract.
const (
	IncidentOpen     = "open"
	IncidentFixing   = "fixing"
	IncidentResolved = "resolved"
	IncidentMuted    = "muted"
)

// Incident is one problem the Pocket AI Agent is watching: a crashing error
// group or a dead letter pile of webhooks.
type Incident struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"projectId"`
	Title          string     `json:"title"`
	PlainSummary   string     `json:"plainSummary"`
	FilePath       string     `json:"filePath"`
	CommitSHA      string     `json:"commitSha"`
	Status         string     `json:"status"`
	PausedCount    int        `json:"pausedCount"`
	PRURL          string     `json:"prUrl"`
	Origin         string     `json:"-"`
	OriginRef      string     `json:"-"`
	MuteUntil      *time.Time `json:"-"`
	LastAlertPhone string     `json:"-"`
	LastAlertAt    *time.Time `json:"-"`
	LastSeenAt     time.Time  `json:"lastSeenAt"`
	CreatedAt      time.Time  `json:"-"`
}

// Incident origins.
const (
	OriginErrorSpike        = "error_spike"
	OriginWebhookDeadLetter = "webhook_dead_letter"
)

// ErrorGroup buckets repeated crashes with the same fingerprint.
type ErrorGroup struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"projectId"`
	Fingerprint     string    `json:"fingerprint"`
	Title           string    `json:"title"`
	FilePath        string    `json:"filePath"`
	Line            int       `json:"line"`
	EventCount      int       `json:"eventCount"`
	WindowCount     int       `json:"windowCount"`
	WindowStartedAt time.Time `json:"windowStartedAt"`
	IncidentID      string    `json:"incidentId"`
	FirstSeenAt     time.Time `json:"firstSeenAt"`
	LastSeenAt      time.Time `json:"lastSeenAt"`
}

// ErrorEvent is one crash report sent by the SDK.
type ErrorEvent struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	GroupID     string    `json:"groupId"`
	Fingerprint string    `json:"-"`
	Message     string    `json:"message"`
	Stack       string    `json:"stack"`
	FilePath    string    `json:"filePath"`
	Line        int       `json:"line"`
	DeliveryID  string    `json:"deliveryId"`
	CreatedAt   time.Time `json:"createdAt"`
}

// DeliveryFilter narrows delivery listings. WorkspaceID scopes the listing to
// one workspace; background workers leave it empty to sweep every workspace.
type DeliveryFilter struct {
	WorkspaceID string
	ProjectID   string
	Status      string
	IncidentID  string
	Limit       int
}

// IncidentFilter narrows incident listings. WorkspaceID scopes the listing to
// one workspace; background workers leave it empty to sweep every workspace.
type IncidentFilter struct {
	WorkspaceID string
	ProjectID   string
	AlertPhone  string
	Limit       int
}

// Workspace is one tenant boundary. Projects, sessions, and users all hang off
// workspaces so two teams can never see each other's data.
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerUserID string    `json:"ownerUserId"`
	CreatedAt   time.Time `json:"createdAt"`
}

// User is one GitHub account that can sign in to PockDev.
type User struct {
	ID                 string    `json:"id"`
	GitHubID           int64     `json:"githubId"`
	GitHubLogin        string    `json:"githubLogin"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	AvatarURL          string    `json:"avatarUrl"`
	DefaultWorkspaceID string    `json:"defaultWorkspaceId"`
	CreatedAt          time.Time `json:"createdAt"`
}

// WorkspaceSummary is one workspace membership row for the workspace switcher.
type WorkspaceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

// Session is one issued login token. Only the SHA-256 hash of the raw token is
// ever stored; the raw token lives in the browser fragment alone.
type Session struct {
	TokenHash   string    `json:"-"`
	UserID      string    `json:"userId"`
	WorkspaceID string    `json:"workspaceId"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
}
