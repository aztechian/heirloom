package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/aztechian/heirloom/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	applyRoutes(mux, &storage.LocalStorage{BasePath: t.TempDir()})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postCollection(t *testing.T, srv *httptest.Server, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(srv.URL+"/api/v1/collections", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decodeProblem(t *testing.T, resp *http.Response) types.Problem {
	t.Helper()
	var p types.Problem
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&p))
	return p
}

func TestCreateCollectionHTTP_Success(t *testing.T) {
	srv := newTestServer(t)

	resp := postCollection(t, srv, `{"name":"My Photo Album"}`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var got types.Collection
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "my-photo-album", got.Slug)
}

func TestCreateCollectionHTTP_ValidationEnforcedByMiddleware(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{name: "missing required name", body: `{}`},
		{name: "name too long", body: `{"name":"` + strings.Repeat("a", 201) + `"}`},
		{name: "slug with invalid characters", body: `{"name":"ok","slug":"Not_Valid!"}`},
		{name: "slug too long", body: `{"name":"ok","slug":"` + strings.Repeat("a", 65) + `"}`},
		{name: "description too long", body: `{"name":"ok","description":"` + strings.Repeat("a", 2001) + `"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := postCollection(t, srv, tc.body)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

			p := decodeProblem(t, resp)
			assert.Equal(t, int32(http.StatusUnprocessableEntity), p.Status)
			require.NotNil(t, p.Errors)
			assert.NotEmpty(t, *p.Errors)
		})
	}
}

func TestCreateCollectionHTTP_MalformedRequestIs400(t *testing.T) {
	srv := newTestServer(t)

	resp := postCollection(t, srv, `{not valid json`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	p := decodeProblem(t, resp)
	assert.Equal(t, int32(http.StatusBadRequest), p.Status)
}
