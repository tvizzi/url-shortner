package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/urlservice"
	"url-shortener/internal/storage"
)

func doRequest(t *testing.T, body string, saverMock save.URLSaver) *httptest.ResponseRecorder {
	t.Helper()

	handler := save.New(slogdiscard.NewDiscardLogger(), saverMock)

	req, err := http.NewRequest(http.MethodPost, "/url", bytes.NewReader([]byte(body)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) save.Response {
	t.Helper()

	var resp save.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	return resp
}

func TestSave_Success(t *testing.T) {
	t.Run("explicit alias", func(t *testing.T) {
		urlSaverMock := mocks.NewURLSaver(t)
		urlSaverMock.On("SaveURL", mock.Anything, "https://google.com", "test_alias").
			Return("test_alias", nil).
			Once()

		rr := doRequest(t, `{"url": "https://google.com", "alias": "test_alias"}`, urlSaverMock)

		require.Equal(t, http.StatusCreated, rr.Code)

		resp := decodeResponse(t, rr)
		require.Equal(t, "OK", resp.Status)
		require.Equal(t, "test_alias", resp.Alias)
	})

	t.Run("generated alias", func(t *testing.T) {
		urlSaverMock := mocks.NewURLSaver(t)
		urlSaverMock.On("SaveURL", mock.Anything, "https://google.com", "").
			Return("random123", nil).
			Once()

		rr := doRequest(t, `{"url": "https://google.com"}`, urlSaverMock)

		require.Equal(t, http.StatusCreated, rr.Code)

		resp := decodeResponse(t, rr)
		require.Equal(t, "OK", resp.Status)
		require.NotEmpty(t, resp.Alias)
	})
}

func TestSave_BadRequest(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "truncated JSON",
			body: `{"url": "https://google.com", "alias":`,
		},
		{
			name: "not JSON at all",
			body: `this is definitely not json`,
		},
		{
			name: "empty body",
			body: ``,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlSaverMock := mocks.NewURLSaver(t)

			rr := doRequest(t, tc.body, urlSaverMock)

			require.Equal(t, http.StatusBadRequest, rr.Code)

			resp := decodeResponse(t, rr)
			require.Equal(t, "Error", resp.Status)
		})
	}
}

func TestSave_ValidationError(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		respError string
	}{
		{
			name:      "empty URL",
			url:       "",
			respError: "field URL is a required field",
		},
		{
			name:      "invalid URL",
			url:       "not a valid url",
			respError: "field URL is not a valid URL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlSaverMock := mocks.NewURLSaver(t)
			message := "field URL is not a valid URL"
			if tc.url == "" {
				message = "field URL is a required field"
			}
			urlSaverMock.On("SaveURL", mock.Anything, tc.url, "some_alias").Return("", urlservice.ValidationError{Message: message}).Once()

			body := `{"url": "` + tc.url + `", "alias": "some_alias"}`
			rr := doRequest(t, body, urlSaverMock)

			require.Equal(t, http.StatusBadRequest, rr.Code)

			resp := decodeResponse(t, rr)
			require.True(t, strings.Contains(resp.Error, tc.respError))
		})
	}
}

func TestSave_Conflict(t *testing.T) {
	urlSaverMock := mocks.NewURLSaver(t)
	urlSaverMock.On("SaveURL", mock.Anything, "https://google.com", "taken_alias").
		Return("", storage.ErrURLExists).
		Once()

	rr := doRequest(t, `{"url": "https://google.com", "alias": "taken_alias"}`, urlSaverMock)

	require.Equal(t, http.StatusConflict, rr.Code)

	resp := decodeResponse(t, rr)
	require.Equal(t, "Error", resp.Status)
	require.Empty(t, resp.Alias)
}

func TestSave_InternalError(t *testing.T) {
	urlSaverMock := mocks.NewURLSaver(t)
	urlSaverMock.On("SaveURL", mock.Anything, "https://google.com", mock.AnythingOfType("string")).
		Return("", errors.New("unexpected db error")).
		Once()

	rr := doRequest(t, `{"url": "https://google.com"}`, urlSaverMock)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	resp := decodeResponse(t, rr)
	require.Equal(t, "Error", resp.Status)
}
