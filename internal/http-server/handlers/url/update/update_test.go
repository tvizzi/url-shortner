package update_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/update"
	"url-shortener/internal/http-server/handlers/url/update/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func doRequest(t *testing.T, body string, updateMock update.UpdateURL) *httptest.ResponseRecorder {
	t.Helper()

	handler := update.New(slogdiscard.NewDiscardLogger(), updateMock)

	req, err := http.NewRequest(http.MethodPut, "/url", bytes.NewReader([]byte(body)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

// TestUpdate_Success checks a known alias is updated and returns 200 with
// countUpdated > 0.
func TestUpdate_Success(t *testing.T) {
	updateMock := mocks.NewUpdateURL(t)
	updateMock.On("UpdateURL", "test_alias", "https://updated.example.com").
		Return(int64(1), nil).
		Once()

	rr := doRequest(t, `{"alias": "test_alias", "url": "https://updated.example.com"}`, updateMock)

	require.Equal(t, http.StatusOK, rr.Code)

	var body update.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Status)
	require.Equal(t, int64(1), body.CountUpdated)
}

// TestUpdate_BadRequest checks genuinely malformed JSON returns 400 and the
// storage layer is never reached.
func TestUpdate_BadRequest(t *testing.T) {
	updateMock := mocks.NewUpdateURL(t) // no .On(...) => must not be called

	rr := doRequest(t, `{"alias": "test_alias", "url":`, updateMock)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestUpdate_ValidationError checks structurally valid JSON with an invalid
// URL also returns 400 without touching storage.
func TestUpdate_ValidationError(t *testing.T) {
	updateMock := mocks.NewUpdateURL(t) // must not be called

	rr := doRequest(t, `{"alias": "test_alias", "url": "not a valid url"}`, updateMock)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestUpdate_NotFound checks updating an unknown alias returns 404.
func TestUpdate_NotFound(t *testing.T) {
	updateMock := mocks.NewUpdateURL(t)
	updateMock.On("UpdateURL", "missing_alias", "https://updated.example.com").
		Return(int64(0), storage.ErrURLNotFound).
		Once()

	rr := doRequest(t, `{"alias": "missing_alias", "url": "https://updated.example.com"}`, updateMock)

	require.Equal(t, http.StatusNotFound, rr.Code)
}
