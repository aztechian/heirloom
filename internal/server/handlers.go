package server

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/google/uuid"
)

const (
	MaxNameLength        = 256
	MaxSlugLength        = 64
	MaxDescriptionLength = 2000
)

// apiHandlers implements types.StrictServerInterface. Storage, so every
// method below except CreateCollection is a placeholder until a persistence
// layer exists to back it.
type apiHandlers struct{}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// slugify derives a URL-safe slug from a collection name: lowercased, with
// runs of anything other than a-z0-9 collapsed to a single hyphen, and
// leading/trailing hyphens trimmed.
func slugify(name string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case !prevHyphen && b.Len() > 0:
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func problem(status int, title, detail string) types.Problem {
	return types.Problem{
		Status: int32(status),
		Title:  title,
		Detail: &detail,
	}
}

func (apiHandlers) CreateCollection(_ context.Context, request types.CreateCollectionRequestObject) (types.CreateCollectionResponseObject, error) {
	if request.Body == nil {
		return types.CreateCollection400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: types.BadRequestApplicationProblemPlusJSONResponse(
				problem(http.StatusBadRequest, "Missing request body", "A JSON body is required."),
			),
		}, nil
	}

	name := strings.TrimSpace(request.Body.Name)
	if len(name) < 1 || len(name) > MaxNameLength {
		return types.CreateCollection422ApplicationProblemPlusJSONResponse{
			ValidationFailedApplicationProblemPlusJSONResponse: types.ValidationFailedApplicationProblemPlusJSONResponse(
				problem(http.StatusUnprocessableEntity, "Validation failed", "name must be between 1 and 256 characters."),
			),
		}, nil
	}

	slug := slugify(name)
	if request.Body.Slug != nil {
		slug = *request.Body.Slug
	}
	if len(slug) < 1 || len(slug) > MaxSlugLength || !slugPattern.MatchString(slug) {
		return types.CreateCollection422ApplicationProblemPlusJSONResponse{
			ValidationFailedApplicationProblemPlusJSONResponse: types.ValidationFailedApplicationProblemPlusJSONResponse(
				problem(http.StatusUnprocessableEntity, "Validation failed", "slug must match ^[a-z0-9]+(?:-[a-z0-9]+)*$ and be 1-64 characters."),
			),
		}, nil
	}

	if request.Body.Description != nil && len(*request.Body.Description) > MaxDescriptionLength {
		return types.CreateCollection422ApplicationProblemPlusJSONResponse{
			ValidationFailedApplicationProblemPlusJSONResponse: types.ValidationFailedApplicationProblemPlusJSONResponse(
				problem(http.StatusUnprocessableEntity, "Validation failed", "description must be at most 2000 characters."),
			),
		}, nil
	}

	// TODO: persist via the store layer once it exists, mapping a UNIQUE(slug)
	// violation to CreateCollection409ApplicationProblemPlusJSONResponse.

	// Use the S3 CreateNewCollection function to persist the collection once the store layer exists
	// Maybe write the Collection struct as sidecar metadata alongside the S3 object.
	now := time.Now().UTC()
	collection := types.Collection{
		Id:          uuid.New(),
		Name:        name,
		Slug:        slug,
		Description: request.Body.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return types.CreateCollection201JSONResponse{
		Body: collection,
		Headers: types.CreateCollection201ResponseHeaders{
			Location: "/api/v1/collections/" + collection.Id.String(),
		},
	}, nil
}

func (apiHandlers) ListCollections(_ context.Context, _ types.ListCollectionsRequestObject) (types.ListCollectionsResponseObject, error) {
	return types.ListCollectionsdefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "listCollections is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (apiHandlers) GetCollection(_ context.Context, _ types.GetCollectionRequestObject) (types.GetCollectionResponseObject, error) {
	return types.GetCollectiondefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "getCollection is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (apiHandlers) ListAssets(_ context.Context, _ types.ListAssetsRequestObject) (types.ListAssetsResponseObject, error) {
	return types.ListAssetsdefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "listAssets is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (apiHandlers) UploadAsset(_ context.Context, _ types.UploadAssetRequestObject) (types.UploadAssetResponseObject, error) {
	return types.UploadAssetdefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "uploadAsset is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (apiHandlers) GetAsset(_ context.Context, _ types.GetAssetRequestObject) (types.GetAssetResponseObject, error) {
	return types.GetAssetdefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "getAsset is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (apiHandlers) UpdateAsset(_ context.Context, _ types.UpdateAssetRequestObject) (types.UpdateAssetResponseObject, error) {
	return types.UpdateAssetdefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "updateAsset is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}

func (apiHandlers) GetInstance(_ context.Context, _ types.GetInstanceRequestObject) (types.GetInstanceResponseObject, error) {
	return types.GetInstancedefaultApplicationProblemPlusJSONResponse{
		Body:       problem(http.StatusNotImplemented, "Not implemented", "getInstance is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}
