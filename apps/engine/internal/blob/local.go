package blob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Local keeps raw bodies under a base directory on disk.
type Local struct {
	base string
}

// NewLocal creates the base directory if needed.
func NewLocal(base string) (*Local, error) {
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return &Local{base: base}, nil
}

func (l *Local) path(key string) (string, error) {
	clean, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	return filepath.Join(l.base, filepath.FromSlash(clean)), nil
}

// Base returns the root directory of this store.
func (l *Local) Base() string {
	return l.base
}

// Put writes the body to disk.
func (l *Local) Put(_ context.Context, key string, data []byte) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Get reads the body from disk.
func (l *Local) Get(_ context.Context, key string) ([]byte, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read blob %s: %w", key, err)
	}
	return data, nil
}
