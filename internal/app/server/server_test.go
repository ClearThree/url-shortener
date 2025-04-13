package server

import (
	"bytes"
	"compress/gzip"
	"github.com/clearthree/url-shortener/internal/app/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testRequest(t *testing.T, testServer *httptest.Server, method string, path string, contentType string, payload string) (*http.Response, string) {
	var body io.Reader
	if payload != "" {
		body = strings.NewReader(payload)
	} else {
		body = nil
	}
	request, err := http.NewRequest(method, testServer.URL+path, body)
	require.NoError(t, err)
	request.Header.Add("Content-Type", contentType)
	resp, err := testServer.Client().Do(request)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestRouter(t *testing.T) {
	testServer := httptest.NewServer(ShortenURLRouter(nil))
	defer testServer.Close()

	var tests = []struct {
		URL         string
		method      string
		payload     string
		contentType string
		want        string
		status      int
		preCreate   bool
	}{
		{
			URL:         "/",
			method:      http.MethodPost,
			payload:     "https://ya.ru",
			contentType: "text/plain",
			want:        "http://localhost",
			status:      http.StatusCreated,
			preCreate:   false,
		},
		{
			URL:         "/lele",
			method:      http.MethodGet,
			payload:     "",
			contentType: "text/plain",
			want:        "Short url not found",
			status:      http.StatusNotFound,
			preCreate:   false,
		},
		{
			URL:         "/",
			method:      http.MethodGet,
			payload:     "https://ya.ru",
			contentType: "text/plain",
			want:        "",
			status:      http.StatusOK,
			preCreate:   true,
		},
		{
			URL:         "/",
			method:      http.MethodGet,
			payload:     "",
			contentType: "text/plain",
			want:        "",
			status:      http.StatusMethodNotAllowed,
			preCreate:   false,
		},
		{
			URL:         "/api/shorten",
			method:      http.MethodPost,
			payload:     `{"url": "https://ya.ru"}`,
			contentType: "application/json",
			want:        `{"result":"http://localhost`,
			status:      http.StatusCreated,
			preCreate:   false,
		},
		{
			URL:         "/api/shorten",
			method:      http.MethodPut,
			payload:     `{"url": "https://ya.ru"}`,
			contentType: "application/json",
			want:        "",
			status:      http.StatusMethodNotAllowed,
			preCreate:   false,
		},
		{
			URL:         "/api/shorten/batch",
			method:      http.MethodPost,
			payload:     `[{"original_url": "https://ya.ru", "correlation_id": "lelele"}]`,
			contentType: "application/json",
			want:        `"correlation_id":"lelele"`,
			status:      http.StatusCreated,
			preCreate:   false,
		},
		{
			URL:         "/api/shorten/batch",
			method:      http.MethodPut,
			payload:     `[{"original_url": "https://ya.ru", "correlation_id": "lelele"}, {"original_url": "https://yandex.ru", "correlation_id": "lololo"}]`,
			contentType: "application/json",
			want:        "",
			status:      http.StatusMethodNotAllowed,
			preCreate:   false,
		},
	}
	for _, test := range tests {
		var URL = test.URL
		if test.preCreate {
			preResponse, shortURL := testRequest(t, testServer, http.MethodPost, test.URL, test.contentType, test.payload)
			preResponse.Body.Close()
			URL = strings.TrimPrefix(shortURL, "http://localhost:8080")
		}
		resp, got := testRequest(t, testServer, test.method, URL, test.contentType, test.payload)
		resp.Body.Close()
		assert.Equal(t, test.status, resp.StatusCode)
		assert.Contains(t, got, test.want)
	}
}

func TestCompression(t *testing.T) {
	testServer := httptest.NewServer(ShortenURLRouter(nil))
	defer testServer.Close()
	requestBody := `{"url": "https://ya.ru"}`

	t.Run("gzip_sending", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		gzipWriter := gzip.NewWriter(buf)
		_, err := gzipWriter.Write([]byte(requestBody))
		require.NoError(t, err)
		err = gzipWriter.Close()
		require.NoError(t, err)

		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/api/shorten", buf)
		require.NoError(t, err)
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "")
		request.Header.Set("Content-Type", "application/json")

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
	})

	t.Run("gzip_receiving", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/api/shorten", buf)
		require.NoError(t, err)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")
		request.RequestURI = ""

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		gzipReader, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		_, err = io.ReadAll(gzipReader)
		require.NoError(t, err)

	})
}

func TestAuth(t *testing.T) {
	testServer := httptest.NewServer(ShortenURLRouter(nil))
	defer testServer.Close()
	requestBody := "https://ya.ru"

	t.Run("without_token_we_receive_it", func(t *testing.T) {
		body := strings.NewReader(requestBody)
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/", body)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Header.Get("Set-Cookie"))
	})

	t.Run("with_expired_token_we_receive_new_one", func(t *testing.T) {
		body := strings.NewReader(requestBody)
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/", body)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		request.AddCookie(&http.Cookie{
			Name:  middlewares.AuthCookieName,
			Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjbGVhcnRocmVlIiwiZXhwIjoxNzQ0NjU5ODI1LCJpYXQiOjE3NDQ2NTk4MjQsInVzZXJfaWQiOiJkYWNlNGQ2OC0yZjk2LTQzODMtYTYwZC0xNzZiYjAzOWQ4NzUifQ.1YUODLYIoH--rLXLNcm9NmMoI1fS5fCvGPt-ktS4ot4",
		})

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Header.Get("Set-Cookie"))

	})

	t.Run("with_wrong_token_we_receive_401", func(t *testing.T) {
		body := strings.NewReader(requestBody)
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/", body)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		request.AddCookie(&http.Cookie{
			Name:  middlewares.AuthCookieName,
			Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjbGVhcnRocmVlIiwiZXhwIjoxNzQ1MDA1NDI0LCJpYXQiOjE3NDQ2NTk4MjR9.vyxJJPY8CknChfzFIM8maWgZcDIPqvRN6CuNm76bq7Y",
		})

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		defer resp.Body.Close()
	})
	t.Run("some_error_with_token", func(t *testing.T) {
		body := strings.NewReader(requestBody)
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/", body)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		request.AddCookie(&http.Cookie{
			Name:  middlewares.AuthCookieName,
			Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjbGVhcnRocmVlIiwiZXhwIjoxODk0NjU5ODI0LCJpYXQiOjE4OTQ2NTk4MjQsInVzZXJfaWQiOiJkYWNlNGQ2OC0yZjk2LTQzODMtYTYwZC0xNzZiYjAzOWQ4NzUifQ.AFcFQc-LucJrSIirCq4RoAMa9jELOLoWECU53mLhZzY",
		})

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		defer resp.Body.Close()
	})

	t.Run("token_is_ok", func(t *testing.T) {
		body := strings.NewReader(requestBody)
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/", body)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")

		resp, err := testServer.Client().Do(request)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Header.Get("Set-Cookie"))

		request, err = http.NewRequest(http.MethodGet, testServer.URL+"/api/user/urls", nil)
		require.NoError(t, err)
		token := strings.TrimPrefix(resp.Header.Get("Set-Cookie"), middlewares.AuthCookieName+"=")
		token = strings.TrimSuffix(token, "; Path=/")
		require.NotEmpty(t, token)
		request.AddCookie(&http.Cookie{
			Name:  middlewares.AuthCookieName,
			Value: token,
		})
		resp, err = testServer.Client().Do(request)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
