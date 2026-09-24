package storage

import (
	"context"
	"os"
	"path/filepath"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/rs/zerolog"
)

const DefaultCollectionPerms os.FileMode = 0750
const DefaultMetadataPerms os.FileMode = 0640

type LocalStorage struct {
	Logger   *zerolog.Logger
	BasePath string
	MetaPath string
}

func NewLocalStorage(ctx context.Context, basePath string) *LocalStorage {
	storage := &LocalStorage{
		Logger:   zerolog.Ctx(ctx),
		BasePath: basePath,
		MetaPath: basePath + "/" + DefaultMetadataDir,
	}
	storage.initializeBasePath()
	storage.initializeBaseMetadata()

	return storage
}

func (l *LocalStorage) CollectionExists(ctx context.Context, name string) bool {
	if err := ctx.Err(); err != nil {
		return false
	}
	// Logic to check if a collection exists in local storage goes here
	// For example, check if the directory exists
	path := filepath.Join(l.BasePath, name)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		l.Logger.Debug().Str("path", path).Msg("Collection exists")
		return true
	}

	return false
}

func (l *LocalStorage) CreateCollection(ctx context.Context, collection types.Collection) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path := filepath.Join(l.BasePath, collection.Slug)
	if err := os.MkdirAll(path, DefaultCollectionPerms); err != nil {
		return err
	}
	// Create the metadata directory for the collection as well.
	if err := os.MkdirAll(filepath.Join(l.MetaPath, collection.Slug), DefaultCollectionPerms); err != nil {
		return err
	}
	l.Logger.Info().Str("collection", collection.Slug).Msg("Created directory for collection")

	return l.writeCollectionMetadata(collection)
}

func (l *LocalStorage) DeleteCollection(ctx context.Context, collection types.Collection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(l.BasePath, collection.Slug)
	l.Logger.Debug().Str("path", path).Msg("Deleting collection")
	// Delete the metadata associated with the collection before removing the collection itself.
	if err := l.deleteCollectionMetadata(collection); err != nil {
		l.Logger.Error().Err(err).Str("collection", collection.Slug).Msg("Failed to delete collection metadata")
	}

	return os.RemoveAll(path)
}

func (l *LocalStorage) ListCollections(ctx context.Context) []types.Collection {
	if err := ctx.Err(); err != nil {
		return nil
	}
	// Logic to list collections in local storage goes here
	entries, err := os.ReadDir(l.BasePath)
	if err != nil {
		return nil
	}

	collections := make([]types.Collection, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() == DefaultMetadataDir {
				continue
			}
			if collection, err := l.readCollectionMetadata(entry.Name()); err == nil {
				collections = append(collections, collection)
				continue
			}
			l.Logger.Warn().Str("collection", entry.Name()).Msg("Failed to read collection metadata, skipping")
		}
	}

	return collections
}

func (l *LocalStorage) initializeBasePath() string {
	l.Logger.Debug().Str("path", l.BasePath).Msg("Initializing base path")
	if err := os.MkdirAll(l.BasePath, DefaultCollectionPerms); err != nil {
		panic(err)
	}

	return l.BasePath
}

func (l *LocalStorage) initializeBaseMetadata() {
	l.Logger.Debug().Str("path", l.MetaPath).Msg("Initializing base metadata")
	if err := os.MkdirAll(l.MetaPath, DefaultCollectionPerms); err != nil {
		panic(err)
	}
}

func (l *LocalStorage) writeCollectionMetadata(collection types.Collection) error {
	path := filepath.Join(l.MetaPath, collection.Slug+".json")
	l.Logger.Debug().Str("path", path).Msg("Writing collection metadata")
	data, err := types.Marshal(collection)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, DefaultMetadataPerms)
}

func (l *LocalStorage) readCollectionMetadata(slug string) (types.Collection, error) {
	path := filepath.Join(l.MetaPath, slug+".json")
	l.Logger.Debug().Str("path", path).Msg("Reading collection metadata")
	//nolint:gosec // path is derived from a collection slug validated by API middleware, not raw user input
	data, err := os.ReadFile(path)
	if err != nil {
		return types.Collection{}, err
	}

	var col types.Collection
	if col, err = types.Unmarshal[types.Collection](data); err != nil {
		return types.Collection{}, err
	}
	col.Slug = slug

	return col, nil
}

func (l *LocalStorage) deleteCollectionMetadata(collection types.Collection) error {
	dirPath := filepath.Join(l.MetaPath, collection.Slug)
	if err := os.RemoveAll(dirPath); err != nil {
		l.Logger.Error().Err(err).Str("path", dirPath).Msg("Failed to delete collection metadata directory")
	}

	path := filepath.Join(l.MetaPath, collection.Slug+".json")
	l.Logger.Debug().Str("path", path).Msg("Deleting collection metadata")

	return os.Remove(path)
}
