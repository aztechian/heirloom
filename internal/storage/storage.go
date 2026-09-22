package storage

import (
	"context"
	"fmt"

	"github.com/aztechian/heirloom/internal/config"
)

type Storage interface {
	CollectionExists(ctx context.Context, name string) bool
	CreateCollection(ctx context.Context, name string) error
	DeleteCollection(ctx context.Context, name string) error
}

// New selects and constructs the Storage implementation described by cfg.
// A configured S3 bucket takes precedence; otherwise storage falls back to
// the local filesystem rooted at cfg.Server.DataPath.
func New(ctx context.Context, cfg config.Config) (Storage, error) {
	if cfg.S3.Bucket != "" {
		s3Storage := NewS3Storage(ctx, cfg)
		if s3Storage == nil {
			return nil, fmt.Errorf("failed to initialize S3 storage")
		}
		return s3Storage, nil
	}
	return NewLocalStorage(cfg.Server.DataPath), nil
}
