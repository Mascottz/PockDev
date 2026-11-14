package blob

import (
	"context"
	"log"
	"time"
)

// Spool keeps every body on local disk first so ingest stays fast and never
// drops a webhook, then mirrors bodies to a remote store (S3) in the
// background when one is configured.
type Spool struct {
	local   *Local
	remote  Store
	uploads chan uploadJob
}

type uploadJob struct {
	key  string
	data []byte
}

// NewSpool builds the store. remote may be nil for local only setups.
func NewSpool(local *Local, remote Store) *Spool {
	s := &Spool{local: local, remote: remote}
	if remote != nil {
		s.uploads = make(chan uploadJob, 256)
		go s.uploadLoop()
	}
	return s
}

// Put writes locally and queues a background mirror upload.
func (s *Spool) Put(ctx context.Context, key string, data []byte) error {
	if err := s.local.Put(ctx, key, data); err != nil {
		return err
	}
	if s.remote != nil {
		select {
		case s.uploads <- uploadJob{key: key, data: append([]byte(nil), data...)}:
		default:
			// Queue is full: the body is still safe on local disk; retry in
			// the background loop by re-queueing after a pause.
			go func() {
				time.Sleep(500 * time.Millisecond)
				select {
				case s.uploads <- uploadJob{key: key, data: data}:
				default:
					log.Printf("spool: drop mirror upload for %s, local copy kept", key)
				}
			}()
		}
	}
	return nil
}

// Get reads from local disk first, then the remote store.
func (s *Spool) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := s.local.Get(ctx, key)
	if err == nil {
		return data, nil
	}
	if s.remote == nil {
		return nil, err
	}
	remoteData, rerr := s.remote.Get(ctx, key)
	if rerr != nil {
		return nil, err
	}
	// Rehydrate the local copy for next time.
	_ = s.local.Put(ctx, key, remoteData)
	return remoteData, nil
}

func (s *Spool) uploadLoop() {
	for job := range s.uploads {
		var err error
		for attempt := 1; attempt <= 5; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err = s.remote.Put(ctx, job.key, job.data)
			cancel()
			if err == nil {
				break
			}
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		if err != nil {
			log.Printf("spool: mirror upload failed for %s: %v (local copy kept)", job.key, err)
		}
	}
}
