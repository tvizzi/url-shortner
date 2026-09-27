package redirect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/redirect/mocks"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func newTestServer(t *testing.T, urlGetterMock redirect.URLGetter) *httptest.Server {
	t.Helper()

	r := chi.NewRouter()
	r.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock))

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return ts
}

func TestRedirect_Success(t *testing.T) {
	const alias = "test_alias"
	const url = "https://www.google.com/"

	urlGetterMock := mocks.NewURLGetter(t)
	urlGetterMock.On("GetURL", alias).Return(url, nil).Once()

	ts := newTestServer(t, urlGetterMock)

	redirectedToURL, err := api.GetRedirect(ts.URL + "/" + alias)
	require.NoError(t, err)
	require.Equal(t, url, redirectedToURL)
}

func TestRedirect_NotFound(t *testing.T) {
	const alias = "missing_alias"

	urlGetterMock := mocks.NewURLGetter(t)
	urlGetterMock.On("GetURL", alias).Return("", storage.ErrURLNotFound).Once()

	ts := newTestServer(t, urlGetterMock)

	resp, err := http.Get(ts.URL + "/" + alias)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "Error", body.Status)
}
