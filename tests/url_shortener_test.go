package tests

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/update"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/random"
)

const (
	host = "localhost:8082"
)

func getAuth(typeField string) string {
	// Загружаем .env файл
	if err := godotenv.Load("../.env"); err != nil {
		panic("Error loading .env file: " + err.Error())
	}

	// Получаем credentials из переменных окружения
	authUser := os.Getenv("AUTH_USER")
	authPassword := os.Getenv("AUTH_PASSWORD")

	if authUser == "" || authPassword == "" {
		panic("AUTH_USER and AUTH_PASSWORD must be set in .env file")
	}

	if typeField == "user" {
		return authUser
	}
	return authPassword

}

func TestURLShortener_HappyPath(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().
		ContainsKey("alias")
}

//nolint:funlen
func TestURLShortener_SaveRedirectRemove(t *testing.T) {
	testCases := []struct {
		name       string
		url        string
		alias      string
		error      string
		wantStatus int
	}{
		{
			name:       "Valid URL",
			url:        gofakeit.URL(),
			alias:      gofakeit.Word() + gofakeit.Word(),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Invalid URL",
			url:        "invalid_url",
			alias:      gofakeit.Word(),
			error:      "field URL is not a valid URL",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Empty Alias",
			url:        gofakeit.URL(),
			alias:      "",
			wantStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := url.URL{
				Scheme: "http",
				Host:   host,
			}

			e := httpexpect.Default(t, u.String())

			// Save
			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth(getAuth("user"), getAuth("password")).
				Expect().Status(tc.wantStatus).
				JSON().Object()

			if tc.error != "" {
				resp.NotContainsKey("alias")

				resp.Value("error").String().IsEqual(tc.error)

				return
			}

			alias := tc.alias

			if tc.alias != "" {
				resp.Value("alias").String().IsEqual(tc.alias)
			} else {
				resp.Value("alias").String().NotEmpty()

				alias = resp.Value("alias").String().Raw()
			}

			// Redirect
			testRedirect(t, alias, tc.url)

			// Remove
			reqDel := e.DELETE("/"+path.Join("url", alias)).
				WithBasicAuth(getAuth("user"), getAuth("password")).
				Expect().Status(http.StatusOK).
				JSON().Object()

			fmt.Println("reqDel", reqDel.Value("countDeleted"))
			reqDel.Value("countDeleted").Number()

		})
	}
}

func TestURLShortener_SaveConflict(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	alias := gofakeit.Word() + gofakeit.Word() + "_conflict"

	e.POST("/url").
		WithJSON(save.Request{URL: gofakeit.URL(), Alias: alias}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusCreated)

	e.POST("/url").
		WithJSON(save.Request{URL: gofakeit.URL(), Alias: alias}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusConflict)

	// cleanup
	e.DELETE("/"+path.Join("url", alias)).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusOK)
}

func TestURLShortener_Update(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	alias := gofakeit.Word() + gofakeit.Word() + "_update"
	originalURL := gofakeit.URL()
	updatedURL := gofakeit.URL()

	e.POST("/url").
		WithJSON(save.Request{URL: originalURL, Alias: alias}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusCreated)

	e.PUT("/url").
		WithJSON(update.Request{Alias: alias, NewURL: updatedURL}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusOK).
		JSON().Object().
		Value("countUpdated").Number().IsEqual(1)

	testRedirect(t, alias, updatedURL)

	e.DELETE("/"+path.Join("url", alias)).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusOK)

	e.PUT("/url").
		WithJSON(update.Request{Alias: alias, NewURL: updatedURL}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusNotFound)
}

func TestURLShortener_RedirectNotFound(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	e.GET("/" + random.NewRandomString(12)).
		Expect().Status(http.StatusNotFound)
}

func TestURLShortener_NoAuth(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{URL: gofakeit.URL()}).
		Expect().Status(http.StatusUnauthorized)
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   alias,
	}

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)

	require.Equal(t, urlToRedirect, redirectedToURL)
}
