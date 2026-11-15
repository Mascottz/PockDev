// Package blob stores raw webhook bodies on local disk or an S3 compatible
// bucket (Rimbra Buckets in production).
package blob

import (
	"context"
	"fmt"
	"path"
	"strings"
)

// Store keeps raw payload bytes under a flat key.
type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	// Delete removes the body. Removing a missing key is not an error so
	// retention sweeps stay idempotent.
	Delete(ctx context.Context, key string) error
}

// CleanKey normalizes a storage key and rejects path traversal.
func CleanKey(key string) (string, error) {
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("bad storage key")
	}
	key = strings.TrimPrefix(path.Clean("/"+key), "/")
	if key == "" || key == "." {
		return "", fmt.Errorf("bad storage key")
	}
	return key, nil
}
