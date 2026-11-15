package worker

import (
	"context"
	"log"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// RetentionSweeper deletes expired sessions and past retention data once an
// hour, plus once shortly after startup.
type RetentionSweeper struct {
	Store store.Store
	Blobs blob.Store
	// Interval between sweeps; defaults to one hour.
	Interval time.Duration
	Now      func() time.Time
	Logf     func(format string, args ...any)
}

func (s *RetentionSweeper) defaults() {
	if s.Interval <= 0 {
		s.Interval = time.Hour
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Logf == nil {
		s.Logf = log.Printf
	}
}

// Sweep runs one retention pass and returns the prune counts.
func (s *RetentionSweeper) Sweep(ctx context.Context) (storageKeys []string, deletedDeliveries, deletedErrors, deletedSessions int, err error) {
	s.defaults()
	storageKeys, deletedDeliveries, deletedErrors, deletedSessions, err = s.Store.PruneExpiredData(ctx, s.Now())
	if err != nil {
		return nil, 0, 0, 0, err
	}
	// Blob deletes are best effort: a missing blob is fine.
	for _, key := range storageKeys {
		if derr := s.Blobs.Delete(ctx, key); derr != nil {
			s.Logf("retention: blob delete failed for %s: %v", key, derr)
		}
	}
	return storageKeys, deletedDeliveries, deletedErrors, deletedSessions, nil
}

// Run sweeps once shortly after startup and then on every tick until ctx is
// done.
func (s *RetentionSweeper) Run(ctx context.Context) {
	s.defaults()
	if keys, nD, nE, nS, err := s.Sweep(ctx); err != nil {
		s.Logf("retention sweep: %v", err)
	} else if nD+nE+nS > 0 {
		s.Logf("retention sweep: deliveries=%d errors=%d sessions=%d blobs=%d", nD, nE, nS, len(keys))
	}

	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if keys, nD, nE, nS, err := s.Sweep(ctx); err != nil {
				s.Logf("retention sweep: %v", err)
			} else if nD+nE+nS > 0 {
				s.Logf("retention sweep: deliveries=%d errors=%d sessions=%d blobs=%d", nD, nE, nS, len(keys))
			}
		}
	}
}
