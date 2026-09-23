package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/aztechian/heirloom/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const MaxSlugLength = 64

// apiHandlers implements types.StrictServerInterface. It holds the Storage
// dependency used to persist collections; every method below except
// CreateCollection is still a placeholder until the rest of the store layer
// exists to back it.
type apiHandlers struct {
	store storage.Storage
}

func newAPIHandlers(store storage.Storage) apiHandlers {
	return apiHandlers{store: store}
}

// slugify derives a URL-safe slug from a collection name: lowercased, with
// runs of anything other than a-z0-9 collapsed to a single hyphen, and
// leading/trailing hyphens trimmed. The result always satisfies the spec's
// slug pattern; only its length still needs checking by the caller.
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
		//nolint:gosec // status is always an http.Status* constant, well within int32 range
		Status: int32(status),
		Title:  title,
		Detail: &detail,
	}
}

func (h apiHandlers) CreateCollection(ctx context.Context, request types.CreateCollectionRequestObject) (types.CreateCollectionResponseObject, error) {
	if request.Body == nil {
		return types.CreateCollection400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: types.BadRequestApplicationProblemPlusJSONResponse(
				problem(http.StatusBadRequest, "Missing request body", "A JSON body is required."),
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
					problem(http.StatusUnprocessableEntity, "Validation failed", "name does not derive a usable slug; provide one explicitly."),
				),
			}, nil
		}
		slug = &derived
	}

	if h.store.CollectionExists(ctx, *slug) {
		return types.CreateCollection409ApplicationProblemPlusJSONResponse(
			problem(http.StatusConflict, "Collection already exists", "A collection with this slug already exists."),
		), nil
	}

	if err := h.store.CreateCollection(ctx, *slug); err != nil {
		return types.CreateCollection500ApplicationProblemPlusJSONResponse{
			InternalErrorApplicationProblemPlusJSONResponse: types.InternalErrorApplicationProblemPlusJSONResponse(
				problem(http.StatusInternalServerError, "Storage error", "Failed to create collection storage."),
			),
		}, nil
	}

	zerolog.Ctx(ctx).Info().Str("collection", *slug).Msg("Created Collection")
	// Maybe write the Collection struct as sidecar metadata alongside the S3 object.
	now := time.Now().UTC()
	collection := types.Collection{
		Id:          uuid.New(),
		Name:        name,
		Slug:        *slug,
		Description: request.Body.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return types.CreateCollection201JSONResponse{
		Body: collection,
		Headers: types.CreateCollection201ResponseHeaders{
			Location: "/api/v1/collections/" + collection.Slug,
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
