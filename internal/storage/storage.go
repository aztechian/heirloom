package storage

import (
	"context"
	"fmt"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/aztechian/heirloom/internal/config"
	"github.com/rs/zerolog"
)

const DefaultMetadataDir = "._meta"

type CollectionStorage interface {
	CollectionExists(ctx context.Context, name string) bool
	CreateCollection(ctx context.Context, collection types.Collection) error
	ListCollections(ctx context.Context) []types.Collection
	DeleteCollection(ctx context.Context, collection types.Collection) error
}

type Storage interface {
	CollectionStorage
}

// New selects and constructs the Storage implementation described by cfg.
// A configured S3 bucket takes precedence; otherwise storage falls back to
// the local filesystem rooted at cfg.Server.DataPath.
func New(ctx context.Context, cfg config.Config) (Storage, error) {
	log := zerolog.Ctx(ctx)
	if cfg.S3.Bucket != "" {
		s3Storage := NewS3Storage(ctx, cfg)
		if s3Storage == nil {
			return nil, fmt.Errorf("failed to initialize S3 storage")
		}
		log.Debug().Str("bucket", cfg.S3.Bucket).Msg("Initialized S3 storage")

		return s3Storage, nil
	}
	log.Debug().Str("location", cfg.Server.DataPath).Msg("Initialized local storage")

	return NewLocalStorage(ctx, cfg.Server.DataPath), nil
}
