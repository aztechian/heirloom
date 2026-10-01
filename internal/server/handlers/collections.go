package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/rs/zerolog"
)

func (h API) CreateCollection(ctx context.Context, request types.CreateCollectionRequestObject) (types.CreateCollectionResponseObject, error) {
	if request.Body == nil {
		return types.CreateCollection400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: types.BadRequestApplicationProblemPlusJSONResponse(
				types.NewProblem(http.StatusBadRequest, "Missing request body", "A JSON body is required."),
			),
		}, nil
	}

	// name/slug/description constraints from the spec (length, pattern) are
	// enforced by request validation middleware before this handler runs.
	name := strings.TrimSpace(request.Body.Name)

	slug := request.Body.Slug
	if slug == nil {
		// A slug derived from name bypasses the middleware's validation of
		// client-supplied slugs, so its length is still this handler's problem.
		derived := slugify(name)
		if len(derived) < 1 || len(derived) > MaxSlugLength {
			return types.CreateCollection422ApplicationProblemPlusJSONResponse{
				ValidationFailedApplicationProblemPlusJSONResponse: types.ValidationFailedApplicationProblemPlusJSONResponse(
					types.NewProblem(http.StatusUnprocessableEntity, "Validation failed", "name does not derive a usable slug; provide one explicitly."),
				),
			}, nil
		}
		slug = &derived
	}

	var assets int64 = 0
	collection := types.Collection{
		Name:        name,
		Slug:        *slug,
		Description: request.Body.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		AssetCount:  &assets,
	}

	if h.collections.CollectionExists(ctx, *slug) {
		return types.CreateCollection409ApplicationProblemPlusJSONResponse(
			types.NewProblem(http.StatusConflict, "Collection already exists", "A collection with this slug already exists."),
		), nil
	}

	if err := h.collections.CreateCollection(ctx, collection); err != nil {
		return types.CreateCollection500ApplicationProblemPlusJSONResponse{
			InternalErrorApplicationProblemPlusJSONResponse: types.InternalErrorApplicationProblemPlusJSONResponse(
				types.NewProblem(http.StatusInternalServerError, "Storage error", "Failed to create collection storage."),
			),
		}, nil
	}
	zerolog.Ctx(ctx).Info().Str("collection", collection.Slug).Msg("Created Collection")

	return types.CreateCollection201JSONResponse{
		Body: collection,
		Headers: types.CreateCollection201ResponseHeaders{
			Location: "/api/v1/collections/" + collection.Slug,
		},
	}, nil
}

func (h API) ListCollections(ctx context.Context, _ types.ListCollectionsRequestObject) (types.ListCollectionsResponseObject, error) {
	collections := h.collections.ListCollections(ctx)
	return types.ListCollections200JSONResponse{
		Collections: collections,
	}, nil
}

func (API) GetCollection(_ context.Context, _ types.GetCollectionRequestObject) (types.GetCollectionResponseObject, error) {
	return types.GetCollectiondefaultApplicationProblemPlusJSONResponse{
		Body:       types.NewProblem(http.StatusNotImplemented, "Not implemented", "getCollection is not implemented yet."),
		StatusCode: http.StatusNotImplemented,
	}, nil
}
