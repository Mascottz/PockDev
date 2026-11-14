package webhook

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// TriggerFunc fires the Pocket AI Agent for a new incident.
type TriggerFunc func(incidentID string)

// DeadLetterHandler reacts when a webhook stops retrying: it writes the schema
// drift note, files (or reuses) the dead letter incident, and wakes the agent.
type DeadLetterHandler struct {
	Store   store.Store
	Trigger TriggerFunc
	Now     func() time.Time
	Logf    func(format string, args ...any)
}

// Handle is the worker OnDeadLetter callback.
func (h *DeadLetterHandler) Handle(ctx context.Context, d *model.Delivery) {
	logf := h.Logf
	if logf == nil {
		logf = log.Printf
	}
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}

	healthy, err := h.Store.RecentDeliveredKeys(ctx, d.ProjectID, 20)
	if err == nil {
		if note := DriftNote(d.PayloadKeys, healthy); note != "" {
			d.SchemaDriftNote = note
			d.UpdatedAt = now
			if err := h.Store.UpdateDelivery(ctx, d); err != nil {
				logf("deadletter: update drift note: %v", err)
			}
		}
	}

	// One incident covers the paused pile per project at a time.
	var inc *model.Incident
	if listed, err := h.Store.ListIncidents(ctx, model.IncidentFilter{ProjectID: d.ProjectID, Limit: 50}); err == nil {
		for i := range listed {
			cand := &listed[i]
			if cand.Origin == model.OriginWebhookDeadLetter &&
				(cand.Status == model.IncidentOpen || cand.Status == model.IncidentFixing) {
				inc = cand
				break
			}
		}
	}

	created := false
	if inc == nil {
		project, err := h.Store.GetProject(ctx, d.ProjectID)
		title := "Webhooks paused"
		if err == nil {
			title = fmt.Sprintf("Webhooks paused for %s", project.Name)
		}
		inc = &model.Incident{
			ID:         id.New("inc_", 24),
			ProjectID:  d.ProjectID,
			Title:      title,
			Origin:     model.OriginWebhookDeadLetter,
			OriginRef:  d.ID,
			Status:     model.IncidentOpen,
			LastSeenAt: now,
			CreatedAt:  now,
		}
		if err := h.Store.CreateIncident(ctx, inc); err != nil {
			logf("deadletter: create incident: %v", err)
			return
		}
		created = true
	}

	if d.LinkedIncidentID != inc.ID {
		d.LinkedIncidentID = inc.ID
		d.UpdatedAt = now
		if err := h.Store.UpdateDelivery(ctx, d); err != nil {
			logf("deadletter: link delivery: %v", err)
		}
	}

	if paused, err := h.Store.ListDeliveries(ctx, model.DeliveryFilter{IncidentID: inc.ID, Status: model.DeliveryPaused}); err == nil {
		inc.PausedCount = len(paused)
	}
	inc.LastSeenAt = now
	if err := h.Store.UpdateIncident(ctx, inc); err != nil {
		logf("deadletter: update incident: %v", err)
	}

	if created && h.Trigger != nil {
		h.Trigger(inc.ID)
	}
}
