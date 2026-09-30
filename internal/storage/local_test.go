package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/aztechian/heirloom/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCollection(slug string) types.Collection {
	return types.Collection{
		Name: slug,
		Slug: slug,
	}
}

func TestNewLocalStorage(t *testing.T) {
	base := t.TempDir()
	ls := storage.NewLocalStorage(context.Background(), base)

	require.NotNil(t, ls)
	assert.Equal(t, base, ls.BasePath)
	assert.Equal(t, base+"/"+storage.DefaultMetadataDir, ls.MetaPath)

	// Base path and metadata directory should both exist after construction.
	info, err := os.Stat(base)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	metaInfo, err := os.Stat(ls.MetaPath)
	require.NoError(t, err)
	assert.True(t, metaInfo.IsDir())
}

func TestNewLocalStorage_PanicsOnUnwritableBase(t *testing.T) {
	// Creating a regular file, then asking for a base path nested under it,
	// forces os.MkdirAll to fail (ENOTDIR), which NewLocalStorage turns into a panic.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))

	assert.Panics(t, func() {
		storage.NewLocalStorage(context.Background(), filepath.Join(blocker, "sub"))
	})
}

func TestLocalStorage_CollectionExists(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())

	assert.False(t, ls.CollectionExists(context.Background(), "nope"))

	require.NoError(t, ls.CreateCollection(context.Background(), newCollection("photos")))
	assert.True(t, ls.CollectionExists(context.Background(), "photos"))
}

func TestLocalStorage_CollectionExists_CanceledContext(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, ls.CollectionExists(ctx, "anything"))
}

func TestLocalStorage_CreateCollection(t *testing.T) {
	base := t.TempDir()
	ls := storage.NewLocalStorage(context.Background(), base)

	col := newCollection("vacation")
	require.NoError(t, ls.CreateCollection(context.Background(), col))

	// Content directory and metadata directory should both exist.
	_, err := os.Stat(filepath.Join(base, "vacation"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(ls.MetaPath, "vacation"))
	require.NoError(t, err)

	// Metadata JSON sidecar file should exist and round-trip via ListCollections.
	_, err = os.Stat(filepath.Join(ls.MetaPath, "vacation.json"))
	require.NoError(t, err)
}

func TestLocalStorage_CreateCollection_CanceledContext(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ls.CreateCollection(ctx, newCollection("photos"))
	assert.Error(t, err)
}

func TestLocalStorage_DeleteCollection(t *testing.T) {
	base := t.TempDir()
	ls := storage.NewLocalStorage(context.Background(), base)

	col := newCollection("temp")
	require.NoError(t, ls.CreateCollection(context.Background(), col))
	require.True(t, ls.CollectionExists(context.Background(), "temp"))

	require.NoError(t, ls.DeleteCollection(context.Background(), col))
	assert.False(t, ls.CollectionExists(context.Background(), "temp"))

	_, err := os.Stat(filepath.Join(ls.MetaPath, "temp.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_DeleteCollection_CanceledContext(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ls.DeleteCollection(ctx, newCollection("photos"))
	assert.Error(t, err)
}

func TestLocalStorage_DeleteCollection_MissingMetadata(t *testing.T) {
	// DeleteCollection only logs metadata-deletion failures; it still
	// succeeds (via os.RemoveAll, which is a no-op on a missing directory)
	// even when the collection was never created.
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())
	err := ls.DeleteCollection(context.Background(), newCollection("never-created"))
	assert.NoError(t, err)
}

func TestLocalStorage_ListCollections(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())

	require.NoError(t, ls.CreateCollection(context.Background(), newCollection("alpha")))
	require.NoError(t, ls.CreateCollection(context.Background(), newCollection("beta")))

	collections := ls.ListCollections(context.Background())
	require.Len(t, collections, 2)

	slugs := []string{collections[0].Slug, collections[1].Slug}
	assert.ElementsMatch(t, []string{"alpha", "beta"}, slugs)
}

func TestLocalStorage_ListCollections_Empty(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())
	assert.Empty(t, ls.ListCollections(context.Background()))
}

func TestLocalStorage_ListCollections_CanceledContext(t *testing.T) {
	ls := storage.NewLocalStorage(context.Background(), t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.Nil(t, ls.ListCollections(ctx))
}

func TestLocalStorage_ListCollections_SkipsUnreadableMetadata(t *testing.T) {
	base := t.TempDir()
	ls := storage.NewLocalStorage(context.Background(), base)

	// A directory that exists in BasePath but has no corresponding metadata
	// sidecar file should be skipped rather than failing the whole listing.
	require.NoError(t, os.MkdirAll(filepath.Join(base, "orphan"), storage.DefaultCollectionPerms))

	assert.Empty(t, ls.ListCollections(context.Background()))
}

func TestLocalStorage_ListCollections_UnreadableBasePath(t *testing.T) {
	base := t.TempDir()
	ls := storage.NewLocalStorage(context.Background(), base)
	require.NoError(t, os.RemoveAll(base))

	assert.Nil(t, ls.ListCollections(context.Background()))
}
