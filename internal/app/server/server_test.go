package server

import (
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
	testServer := httptest.NewServer(ShortenURLRouter())
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
