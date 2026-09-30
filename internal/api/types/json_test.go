package types_test

import (
	"testing"
	"time"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	desc := "a description"
	var count int64 = 5
	col := types.Collection{
		Name:        "My Collection",
		Slug:        "my-collection",
		Description: &desc,
		CreatedAt:   now,
		UpdatedAt:   now,
		AssetCount:  &count,
	}

	data, err := types.Marshal(col)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"slug":"my-collection"`)

	got, err := types.Unmarshal[types.Collection](data)
	require.NoError(t, err)
	assert.Equal(t, col.Slug, got.Slug)
	assert.Equal(t, col.Name, got.Name)
	assert.Equal(t, col.Description, got.Description)
	assert.Equal(t, col.CreatedAt, got.CreatedAt)
	require.NotNil(t, got.AssetCount)
	assert.Equal(t, *col.AssetCount, *got.AssetCount)
}

func TestMarshal_Error(t *testing.T) {
	// channels are not JSON-marshalable, forcing Marshal to return an error.
	_, err := types.Marshal(make(chan int))
	assert.Error(t, err)
}

func TestUnmarshal_Error(t *testing.T) {
	_, err := types.Unmarshal[types.Collection]([]byte(`{not valid json`))
	assert.Error(t, err)
}

func TestUnmarshal_Empty(t *testing.T) {
	got, err := types.Unmarshal[types.Collection]([]byte(`{}`))
	require.NoError(t, err)
	assert.Empty(t, got.Slug)
	assert.Empty(t, got.Name)
}
