package handlers

import (
	"context"
	"net/http"

	"github.com/aztechian/heirloom/internal/api/types"
)

func (API) GetInstance(_ context.Context, _ types.GetInstanceRequestObject) (types.GetInstanceResponseObject, error) {
	return types.GetInstancedefaultApplicationProblemPlusJSONResponse{
		Body:       types.NewProblem(http.StatusNotImplemented, "Not implemented", "getInstance is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}
