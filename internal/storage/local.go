package storage

import (
	"context"
	"os"
)

type LocalStorage struct {
	BasePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	storage := &LocalStorage{
		BasePath: basePath,
	}
	storage.initializeBasePath()
	return storage
}

func (l *LocalStorage) initializeBasePath() string {
	if err := os.MkdirAll(l.BasePath, 0755); err != nil {
		panic(err)
	}
	return l.BasePath
}

func (l *LocalStorage) CollectionExists(ctx context.Context, name string) bool {
	if err := ctx.Err(); err != nil {
		return false
	}
	// Logic to check if a collection exists in local storage goes here
	// For example, check if the directory exists
	path := l.BasePath + "/" + name
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return true
	}
	return false
}

func (l *LocalStorage) CreateCollection(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Logic to create a collection in local storage goes here
	path := l.BasePath + "/" + name
	return os.MkdirAll(path, 0755)
}

func (l *LocalStorage) DeleteCollection(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Logic to delete a collection in local storage goes here
	path := l.BasePath + "/" + name
	return os.RemoveAll(path)
}
