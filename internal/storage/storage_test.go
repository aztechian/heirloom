package storage_test

import (
	"context"
	"testing"

	"github.com/aztechian/heirloom/internal/config"
	"github.com/aztechian/heirloom/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_LocalFallback(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{DataPath: t.TempDir()},
	}

	store, err := storage.New(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, store)

	_, ok := store.(*storage.LocalStorage)
	assert.True(t, ok, "expected *storage.LocalStorage, got %T", store)
}

func TestNew_S3WhenBucketConfigured(t *testing.T) {
	cfg := config.Config{
		S3: config.S3Config{
			Bucket: "my-bucket",
			Region: "us-east-1",
		},
	}

	store, err := storage.New(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, store)

	_, ok := store.(*storage.S3Storage)
	assert.True(t, ok, "expected *storage.S3Storage, got %T", store)
}
