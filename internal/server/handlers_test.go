package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

// fakeStorage is a minimal in-memory storage.Storage used to exercise
// handlers without touching the filesystem or a network-backed store.
type fakeStorage struct {
	existing  map[string]bool
	createErr error
}

func (f *fakeStorage) CollectionExists(_ context.Context, name string) bool {
	return f.existing[name]
}

func (f *fakeStorage) CreateCollection(_ context.Context, collection types.Collection) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.existing == nil {
		f.existing = map[string]bool{}
	}
	f.existing[collection.Slug] = true
	return nil
}

func (f *fakeStorage) DeleteCollection(_ context.Context, collection types.Collection) error {
	delete(f.existing, collection.Slug)
	return nil
}

func (f *fakeStorage) ListCollections(_ context.Context) []types.Collection {
	collections := make([]types.Collection, 0, len(f.existing))
	for slug := range f.existing {
		collections = append(collections, types.Collection{
			Slug:      slug,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Name:      slug,
		})
	}
	return collections
}

func TestCreateCollection_NilBody(t *testing.T) {
	resp, err := (apiHandlers{}).CreateCollection(context.Background(), types.CreateCollectionRequestObject{Body: nil})
	require.NoError(t, err)

	got, ok := resp.(types.CreateCollection400ApplicationProblemPlusJSONResponse)
	require.True(t, ok, "expected 400 response, got %T", resp)
	assert.Equal(t, int32(http.StatusBadRequest), got.Status)
}

func TestCreateCollection_Success(t *testing.T) {
	tests := []struct {
		name    string
		body    types.CreateCollectionRequest
		wantSlg string
	}{
		{
			name:    "derives slug from name",
			body:    types.CreateCollectionRequest{Name: "My Photo Album"},
			wantSlg: "my-photo-album",
		},
		{
			name:    "uses explicit slug",
			body:    types.CreateCollectionRequest{Name: "My Photo Album", Slug: ptr("custom-slug")},
			wantSlg: "custom-slug",
		},
		{
			name:    "trims whitespace from name",
			body:    types.CreateCollectionRequest{Name: "  Padded  "},
			wantSlg: "padded",
		},
		{
			name:    "accepts description",
			body:    types.CreateCollectionRequest{Name: "Vacation", Description: ptr("Trip photos")},
			wantSlg: "vacation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := apiHandlers{store: &fakeStorage{}}
			resp, err := h.CreateCollection(context.Background(), types.CreateCollectionRequestObject{Body: &tc.body})
			require.NoError(t, err)

			got, ok := resp.(types.CreateCollection201JSONResponse)
			require.True(t, ok, "expected 201 response, got %T", resp)

			assert.Equal(t, tc.wantSlg, got.Body.Slug)
			assert.Equal(t, strings.TrimSpace(tc.body.Name), got.Body.Name)
			assert.False(t, got.Body.CreatedAt.IsZero())
			assert.False(t, got.Body.UpdatedAt.IsZero())
			assert.Equal(t, tc.body.Description, got.Body.Description)
			assert.Equal(t, "/api/v1/collections/"+got.Body.Slug, got.Headers.Location)
		})
	}
}

func TestCreateCollection_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body types.CreateCollectionRequest
	}{
		{
			// Every rune is stripped by slugify, leaving nothing to derive from.
			name: "name derives an empty slug",
			body: types.CreateCollectionRequest{Name: "!!!"},
		},
		{
			// slugify never inserts hyphens as densely as MaxSlugLength+1 plain runs,
			// so a name this long reliably derives an over-length slug.
			name: "name derives an over-length slug",
			body: types.CreateCollectionRequest{Name: strings.Repeat("a", MaxSlugLength+1)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := (apiHandlers{}).CreateCollection(context.Background(), types.CreateCollectionRequestObject{Body: &tc.body})
			require.NoError(t, err)

			got, ok := resp.(types.CreateCollection422ApplicationProblemPlusJSONResponse)
			require.True(t, ok, "expected 422 response, got %T", resp)
			assert.Equal(t, int32(http.StatusUnprocessableEntity), got.Status)
		})
	}
}
