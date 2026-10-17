package handlers

import (
	"context"
	"strings"

	"github.com/aztechian/heirloom/internal/api/types"
)

func (h API) GetInstance(_ context.Context, _ types.GetInstanceRequestObject) (types.GetInstanceResponseObject, error) {
	return types.GetInstance200JSONResponse{
		Version:               h.config.Version,
		MaxAssetBytes:         (int64)(h.config.Server.MaxAssetsBytes),
		AcceptedMediaTypes:    strings.Split(h.config.Server.AllowedMimeTypes, ","),
		AuthenticationEnabled: false,
	}, nil
}
