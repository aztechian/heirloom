package handlers

import (
	"context"
	"net/http"

	"github.com/aztechian/heirloom/internal/api/types"
)

func (API) ListAssets(_ context.Context, _ types.ListAssetsRequestObject) (types.ListAssetsResponseObject, error) {
	return types.ListAssetsdefaultApplicationProblemPlusJSONResponse{
		Body:       types.NewProblem(http.StatusNotImplemented, "Not implemented", "listAssets is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (API) UploadAsset(_ context.Context, _ types.UploadAssetRequestObject) (types.UploadAssetResponseObject, error) {
	return types.UploadAssetdefaultApplicationProblemPlusJSONResponse{
		Body:       types.NewProblem(http.StatusNotImplemented, "Not implemented", "uploadAsset is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (API) GetAsset(_ context.Context, _ types.GetAssetRequestObject) (types.GetAssetResponseObject, error) {
	return types.GetAssetdefaultApplicationProblemPlusJSONResponse{
		Body:       types.NewProblem(http.StatusNotImplemented, "Not implemented", "getAsset is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (API) UpdateAsset(_ context.Context, _ types.UpdateAssetRequestObject) (types.UpdateAssetResponseObject, error) {
	return types.UpdateAssetdefaultApplicationProblemPlusJSONResponse{
		Body:       types.NewProblem(http.StatusNotImplemented, "Not implemented", "updateAsset is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}
