package delete_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	del "url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/delete/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func newTestServer(t *testing.T, deleteMock del.DeleteURL) *httptest.Server {
	t.Helper()

	r := chi.NewRouter()
	r.Delete("/url/{alias}", del.New(slogdiscard.NewDiscardLogger(), deleteMock))

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return ts
}

func doDelete(t *testing.T, ts *httptest.Server, alias string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/url/"+alias, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func TestDelete_Success(t *testing.T) {
	const alias = "test_alias"

	deleteMock := mocks.NewDeleteURL(t)
	deleteMock.On("DeleteURL", mock.Anything, alias).Return(int64(1), nil).Once()

	ts := newTestServer(t, deleteMock)

	resp := doDelete(t, ts, alias)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body del.Response
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "OK", body.Status)
	require.Equal(t, int64(1), body.CountDeleted)
}

func TestDelete_NotFound(t *testing.T) {
	const alias = "missing_alias"

	deleteMock := mocks.NewDeleteURL(t)
	deleteMock.On("DeleteURL", mock.Anything, alias).Return(int64(0), storage.ErrURLNotFound).Once()

	ts := newTestServer(t, deleteMock)

	resp := doDelete(t, ts, alias)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body del.Response
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "Error", body.Status)
}
