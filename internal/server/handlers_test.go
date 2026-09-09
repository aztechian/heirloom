package server

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

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
			resp, err := (apiHandlers{}).CreateCollection(context.Background(), types.CreateCollectionRequestObject{Body: &tc.body})
			require.NoError(t, err)

			got, ok := resp.(types.CreateCollection201JSONResponse)
			require.True(t, ok, "expected 201 response, got %T", resp)

			assert.Equal(t, tc.wantSlg, got.Body.Slug)
			assert.Equal(t, strings.TrimSpace(tc.body.Name), got.Body.Name)
			assert.NotEmpty(t, got.Body.Id.String())
			assert.False(t, got.Body.CreatedAt.IsZero())
			assert.False(t, got.Body.UpdatedAt.IsZero())
			assert.Equal(t, tc.body.Description, got.Body.Description)
			assert.Equal(t, "/api/v1/collections/"+got.Body.Id.String(), got.Headers.Location)
		})
	}
}

func TestCreateCollection_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body types.CreateCollectionRequest
	}{
		{
			name: "empty name",
			body: types.CreateCollectionRequest{Name: "   "},
		},
		{
			name: "name too long",
			body: types.CreateCollectionRequest{Name: strings.Repeat("a", MaxNameLength+1)},
		},
		{
			name: "explicit slug too long",
			body: types.CreateCollectionRequest{Name: "ok", Slug: ptr(strings.Repeat("a", MaxSlugLength+1))},
		},
		{
			name: "explicit slug with invalid characters",
			body: types.CreateCollectionRequest{Name: "ok", Slug: ptr("Not_Valid!")},
		},
		{
			name: "explicit empty slug",
			body: types.CreateCollectionRequest{Name: "ok", Slug: ptr("")},
		},
		{
			name: "description too long",
			body: types.CreateCollectionRequest{Name: "ok", Description: ptr(strings.Repeat("a", MaxDescriptionLength+1))},
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
