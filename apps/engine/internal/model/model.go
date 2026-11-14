package model

import "time"

// Project is one app or service being watched by PockDev.
type Project struct {
	ID            string    `json:"id"`
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

// DeliveryFilter narrows delivery listings.
type DeliveryFilter struct {
	ProjectID  string
	Status     string
	IncidentID string
	Limit      int
}

// IncidentFilter narrows incident listings.
type IncidentFilter struct {
	ProjectID  string
	AlertPhone string
	Limit      int
}
