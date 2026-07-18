// Package storage abstracts file storage for uploads (resumes, offer PDFs).
// LocalStorage writes to the filesystem; an S3 driver can implement the same
// interface later without touching callers.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage stores and retrieves files by an opaque key.
type Storage interface {
	Save(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// New builds the configured storage driver. driver is "local" (default) or "s3".
func New(ctx context.Context, driver, localDir, bucket, region, endpoint string) (Storage, error) {
	switch driver {
	case "", "local":
		return NewLocalStorage(localDir), nil
	case "s3":
		return NewS3Storage(ctx, bucket, region, endpoint)
	default:
		return nil, fmt.Errorf("unknown storage driver %q", driver)
	}
}

// LocalStorage stores files under a base directory. Keys use forward slashes
// and are mapped to OS paths.
type LocalStorage struct{ base string }

// NewLocalStorage builds a LocalStorage rooted at base.
func NewLocalStorage(base string) *LocalStorage { return &LocalStorage{base: base} }

func (s *LocalStorage) path(key string) string {
	return filepath.Join(s.base, filepath.FromSlash(key))
}

// Save writes r to the file at key, creating parent directories as needed.
func (s *LocalStorage) Save(_ context.Context, key string, r io.Reader) error {
	p := s.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Open returns a reader for the file at key. The caller must Close it.
func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(s.path(key))
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}
