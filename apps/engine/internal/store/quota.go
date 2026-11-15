package store

import (
	"context"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
)

// MonthStart returns the start of the UTC month containing now.
func MonthStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// MonthlyEventQuotaReached reports whether the workspace hit its effective
// plan monthly event limit. Store errors report false so ingest stays
// available.
func MonthlyEventQuotaReached(ctx context.Context, st Store, workspaceID string, now time.Time) bool {
	wsp, err := st.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return false
	}
	plan := model.PlanFor(model.EffectivePlan(wsp, now), 0)
	_, events, err := st.WorkspaceUsage(ctx, workspaceID, MonthStart(now))
	if err != nil {
		return false
	}
	return events >= plan.MonthlyEventsLimit
}
