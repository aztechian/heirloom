package handlers

import (
	"context"
	"testing"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/stretchr/testify/require"
)

func TestGetInstance(t *testing.T) {
	resp, err := (API{}).GetInstance(context.Background(), types.GetInstanceRequestObject{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}
