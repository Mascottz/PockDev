// Package worker forwards queued webhook deliveries to project targets with
// exponential backoff retries.
package worker

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// DeadLetterFunc is called when a delivery stops retrying (paused or failed).
type DeadLetterFunc func(ctx context.Context, d *model.Delivery)

// Forwarder runs a small worker pool that pushes deliveries to their target.
type Forwarder struct {
	Store          store.Store
	Blobs          blob.Store
	Client         *http.Client
	Workers        int
	Poll           time.Duration
	MaxAttempts    int
	RetryBase      time.Duration
	MaxBackoff     time.Duration
	RequestTimeout time.Duration
	OnDeadLetter   DeadLetterFunc
	Logf           func(format string, args ...any)
}

// defaults fills unset knobs with their production defaults.
func (f *Forwarder) defaults() {
	if f.Client == nil {
		f.Client = &http.Client{Timeout: 15 * time.Second}
	}
	if f.Workers <= 0 {
		f.Workers = 4
	}
	if f.Poll <= 0 {
		f.Poll = 250 * time.Millisecond
	}
	if f.MaxAttempts <= 0 {
		f.MaxAttempts = 8
	}
	if f.RetryBase <= 0 {
		f.RetryBase = 5 * time.Second
	}
	if f.MaxBackoff <= 0 {
		f.MaxBackoff = 15 * time.Minute
	}
	if f.RequestTimeout <= 0 {
		f.RequestTimeout = 15 * time.Second
	}
	if f.Logf == nil {
		f.Logf = log.Printf
	}
}

// Run blocks until ctx is cancelled.
func (f *Forwarder) Run(ctx context.Context) {
	f.defaults()

	jobs := make(chan *model.Delivery, f.Workers*2)
	var wg sync.WaitGroup
	for i := 0; i < f.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range jobs {
				f.handle(ctx, d)
			}
		}()
	}

	ticker := time.NewTicker(f.Poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case <-ticker.C:
			batch, err := f.Store.ClaimDueDeliveries(ctx, time.Now(), 60*time.Second, f.Workers*2)
			if err != nil {
				if ctx.Err() == nil {
					f.Logf("forwarder: claim error: %v", err)
				}
				continue
			}
			for i := range batch {
				select {
				case jobs <- &batch[i]:
				case <-ctx.Done():
					close(jobs)
					wg.Wait()
					return
				}
			}
		}
	}
}

func (f *Forwarder) handle(ctx context.Context, d *model.Delivery) {
	f.defaults()
	project, err := f.Store.GetProject(ctx, d.ProjectID)
	if err != nil {
		f.Logf("forwarder: project %s missing for %s", d.ProjectID, d.ID)
		return
	}
	if project.TargetURL == "" {
		f.finishAttempt(ctx, d, 0, "project has no target URL", false)
		return
	}

	body, err := f.Blobs.Get(ctx, d.StorageKey)
	if err != nil && d.PayloadPreview != "" {
		body = []byte(d.PayloadPreview)
		err = nil
	}
	if err != nil {
		f.finishAttempt(ctx, d, 0, "payload body missing: "+err.Error(), true)
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, f.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, project.TargetURL, bytes.NewReader(body))
	if err != nil {
		f.finishAttempt(ctx, d, 0, "bad target URL: "+err.Error(), false)
		return
	}
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PockDev-Delivery-Id", d.ID)

	resp, err := f.Client.Do(req)
	if err != nil {
		f.finishAttempt(ctx, d, 0, "target unreachable: "+err.Error(), true)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now()
		d.Status = model.DeliveryDelivered
		d.HTTPStatus = resp.StatusCode
		d.AttemptCount++
		d.LastError = ""
		d.NextAttemptAt = now
		d.UpdatedAt = now
		d.DeliveredAt = &now
		if err := f.Store.UpdateDelivery(ctx, d); err != nil {
			f.Logf("forwarder: update %s: %v", d.ID, err)
		}
		return
	}
	retryable := resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusTooManyRequests
	f.finishAttempt(ctx, d, resp.StatusCode, "target returned "+strconv.Itoa(resp.StatusCode), retryable)
}

// finishAttempt records the result of one forwarding attempt. retryable false
// means a permanent failure.
func (f *Forwarder) finishAttempt(ctx context.Context, d *model.Delivery, status int, errMsg string, retryable bool) {
	now := time.Now()
	d.HTTPStatus = status
	d.LastError = errMsg
	d.AttemptCount++
	d.UpdatedAt = now

	if !retryable {
		d.Status = model.DeliveryFailed
		d.NextAttemptAt = now
	} else if d.AttemptCount >= f.MaxAttempts {
		d.Status = model.DeliveryPaused
		d.NextAttemptAt = now
	} else {
		d.Status = model.DeliveryRetrying
		d.NextAttemptAt = now.Add(f.backoff(d.AttemptCount))
	}
	if err := f.Store.UpdateDelivery(ctx, d); err != nil {
		f.Logf("forwarder: update %s: %v", d.ID, err)
		return
	}
	if d.Status == model.DeliveryPaused || d.Status == model.DeliveryFailed {
		if f.OnDeadLetter != nil {
			f.OnDeadLetter(ctx, d)
		}
	}
}

func (f *Forwarder) backoff(attempt int) time.Duration {
	d := f.RetryBase
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= f.MaxBackoff {
			return f.MaxBackoff
		}
	}
	if d > f.MaxBackoff {
		return f.MaxBackoff
	}
	return d
}

// Replay resets a delivery for an immediate fresh attempt.
func Replay(ctx context.Context, st store.Store, id string) (*model.Delivery, error) {
	d, err := st.GetDelivery(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	d.Status = model.DeliveryRetrying
	d.AttemptCount = 0
	d.LastError = ""
	d.NextAttemptAt = now
	d.UpdatedAt = now
	if err := st.UpdateDelivery(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// ReplayAll resets every paused (or failed) delivery linked to an incident.
func ReplayAll(ctx context.Context, st store.Store, incidentID string) (int, error) {
	paused, err := st.ListDeliveries(ctx, model.DeliveryFilter{IncidentID: incidentID, Status: model.DeliveryPaused})
	if err != nil {
		return 0, err
	}
	failed, err := st.ListDeliveries(ctx, model.DeliveryFilter{IncidentID: incidentID, Status: model.DeliveryFailed})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range append(paused, failed...) {
		if _, err := Replay(ctx, st, d.ID); err == nil {
			n++
		}
	}
	return n, nil
}

// ReplayProject resets every paused (or failed) delivery belonging to a project.
func ReplayProject(ctx context.Context, st store.Store, projectID string) (int, error) {
	paused, err := st.ListDeliveries(ctx, model.DeliveryFilter{ProjectID: projectID, Status: model.DeliveryPaused})
	if err != nil {
		return 0, err
	}
	failed, err := st.ListDeliveries(ctx, model.DeliveryFilter{ProjectID: projectID, Status: model.DeliveryFailed})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range append(paused, failed...) {
		if _, err := Replay(ctx, st, d.ID); err == nil {
			n++
		}
	}
	return n, nil
}
