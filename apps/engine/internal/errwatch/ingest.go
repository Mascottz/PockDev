// Package errwatch ingests application crashes reported by the PockDev SDK
// and links them to webhook deliveries.
package errwatch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// TriggerFunc fires the Pocket AI Agent for a new incident.
type TriggerFunc func(incidentID string)

// Ingest handles POST /v1/errors.
type Ingest struct {
	Store          store.Store
	SpikeThreshold int
	Window         time.Duration
	Now            func() time.Time
	OnIncident     TriggerFunc
}

// Payload is the crash report body sent by @pockdev/sdk.
type Payload struct {
	Message    string         `json:"message"`
	Stack      string         `json:"stack"`
	FilePath   string         `json:"filePath"`
	Line       int            `json:"line"`
	Level      string         `json:"level"`
	DeliveryID string         `json:"deliveryId"`
	Context    map[string]any `json:"context"`
}

// ServeHTTP handles POST /v1/errors authenticated with X-PockDev-Key.
func (h *Ingest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := r.Header.Get("X-PockDev-Key")
	if key == "" {
		http.Error(w, "missing X-PockDev-Key", http.StatusUnauthorized)
		return
	}
	project, err := h.Store.GetProjectBySDKKey(r.Context(), key)
	if err != nil {
		http.Error(w, "unknown SDK key", http.StatusUnauthorized)
		return
	}

	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}

	// Monthly event quota for the project workspace.
	if store.MonthlyEventQuotaReached(r.Context(), h.Store, project.WorkspaceID, now) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error": "monthly event quota reached for workspace", "code": "quota_exceeded"}`)
		return
	}

	var p Payload
	if !httpx.DecodeJSON(w, r, &p) {
		return
	}
	if strings.TrimSpace(p.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	window := h.Window
	if window <= 0 {
		window = time.Hour
	}

	ev := &model.ErrorEvent{
		ID:          id.New("err_", 24),
		ProjectID:   project.ID,
		Message:     p.Message,
		Stack:       p.Stack,
		FilePath:    p.FilePath,
		Line:        p.Line,
		DeliveryID:  p.DeliveryID,
		Fingerprint: Fingerprint(project.ID, p.FilePath, p.Line, p.Message),
		CreatedAt:   now,
	}

	group, err := h.Store.RecordErrorEvent(r.Context(), ev, window)
	if err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}

	// Link this exact crash to the exact webhook delivery it came from.
	if ev.DeliveryID != "" {
		if d, err := h.Store.GetDelivery(r.Context(), ev.DeliveryID); err == nil {
			h.linkDelivery(r, d, group, ev, now)
		}
	}

	// A fresh spike opens an incident and wakes the Pocket AI Agent.
	threshold := h.SpikeThreshold
	if threshold <= 0 {
		threshold = 5
	}
	if group.WindowCount >= threshold && group.IncidentID == "" {
		inc := &model.Incident{
			ID:           id.New("inc_", 24),
			ProjectID:    project.ID,
			Title:        firstLine(p.Message),
			PlainSummary: "",
			FilePath:     p.FilePath,
			Origin:       model.OriginErrorSpike,
			OriginRef:    group.ID,
			Status:       model.IncidentOpen,
			LastSeenAt:   now,
			CreatedAt:    now,
		}
		if err := h.Store.CreateIncident(r.Context(), inc); err == nil {
			group.IncidentID = inc.ID
			_ = h.Store.UpdateErrorGroup(r.Context(), group)
			if ev.DeliveryID != "" {
				if d, err := h.Store.GetDelivery(r.Context(), ev.DeliveryID); err == nil {
					if d.LinkedIncidentID == "" {
						d.LinkedIncidentID = inc.ID
						d.UpdatedAt = now
						_ = h.Store.UpdateDelivery(r.Context(), d)
					}
				}
			}
			if h.OnIncident != nil {
				h.OnIncident(inc.ID)
			}
		}
	}

	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"received": true,
		"groupId":  group.ID,
	})
}

// linkDelivery points the delivery at the right incident for this crash. When
// the delivery is dead lettered and has no incident yet, one is created so the
// crash stack and the paused webhooks sit in one place.
func (h *Ingest) linkDelivery(r *http.Request, d *model.Delivery, group *model.ErrorGroup, ev *model.ErrorEvent, now time.Time) {
	if d.LinkedIncidentID != "" {
		return
	}
	incidentID := group.IncidentID
	if incidentID == "" && (d.Status == model.DeliveryPaused || d.Status == model.DeliveryFailed) {
		inc := &model.Incident{
			ID:         id.New("inc_", 24),
			ProjectID:  d.ProjectID,
			Title:      firstLine(ev.Message),
			FilePath:   ev.FilePath,
			Origin:     model.OriginWebhookDeadLetter,
			OriginRef:  d.ID,
			Status:     model.IncidentOpen,
			LastSeenAt: now,
			CreatedAt:  now,
		}
		if err := h.Store.CreateIncident(r.Context(), inc); err == nil {
			incidentID = inc.ID
			if group.IncidentID == "" {
				group.IncidentID = inc.ID
				_ = h.Store.UpdateErrorGroup(r.Context(), group)
			}
			if h.OnIncident != nil {
				h.OnIncident(inc.ID)
			}
		}
	}
	if incidentID != "" {
		d.LinkedIncidentID = incidentID
		d.UpdatedAt = now
		_ = h.Store.UpdateDelivery(r.Context(), d)
	}
}

var noisy = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|0x[0-9a-f]+|\b[0-9a-f]{16,}\b|\b\d+\b)`)

// Fingerprint buckets crashes with the same shape together.
func Fingerprint(projectID, filePath string, line int, message string) string {
	norm := strings.ToLower(noisy.ReplaceAllString(message, "#"))
	h := sha256.Sum256([]byte(projectID + "|" + filePath + "|" + norm))
	return hex.EncodeToString(h[:])[:32]
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}
