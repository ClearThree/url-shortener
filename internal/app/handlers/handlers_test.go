package handlers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCreateShortURLHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name                 string
		requestPayload       string
		requestMethod        string
		requestContentType   string
		requestContentLength string
		want                 want
	}{
		{
			name:                 "Successful short url creation test",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "",
			want: want{
				code:        201,
				response:    `http://localhost:8080/`,
				contentType: "text/plain",
			},
		},
		{
			name:                 "Unsuccessful request due to wrong content-type",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "application/json",
			requestContentLength: "",
			want: want{
				code:        400,
				response:    `Only text/plain content type is allowed`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                 "Unsuccessful request due to wrong content-length",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "писятДва",
			want: want{
				code:        400,
				response:    `Content-Length header is invalid, should be integer`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                 "Unsuccessful request due to too big content",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: strconv.FormatInt(maxPayloadSize+1, 10),
			want: want{
				code:        400,
				response:    `Content is too large`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                 "Unsuccessful request due to empty body",
			requestPayload:       "",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "",
			want: want{
				code:        400,
				response:    `Please provide an url`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                 "Unsuccessful request due to bad request payload",
			requestPayload:       "lelelele",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "",
			want: want{
				code:        400,
				response:    `The provided payload is not a valid URL`,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := strings.NewReader(test.requestPayload)
			request := httptest.NewRequest(test.requestMethod, "/", body)
			request.Header.Add("Content-Type", test.requestContentType)
			if test.requestContentLength != "" {
				request.Header.Add("Content-Length", test.requestContentLength)
			} else {
				request.Header.Add("Content-Length", strconv.FormatInt(request.ContentLength, 10))
			}
			recorder := httptest.NewRecorder()
			CreateShortURLHandler{}.ServeHTTP(recorder, request)

			res := recorder.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Contains(t, string(resBody), test.want.response)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestRedirectToOriginalURLHandler(t *testing.T) {
	type want struct {
		code     int
		response string
		header   string
	}
	tests := []struct {
		name          string
		preloadURL    string
		requestMethod string
		want          want
	}{
		{
			name:          "Successful redirection test",
			preloadURL:    "https://ya.ru",
			requestMethod: http.MethodGet,
			want: want{
				code:     307,
				response: "https://ya.ru",
				header:   "Location",
			},
		},
		{
			name:          "Unsuccessful redirection due to non-existing shortURL test",
			preloadURL:    "",
			requestMethod: http.MethodGet,
			want: want{
				code:     404,
				response: "Short url not found",
				header:   "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var shortURL = "lelelele"
			if test.preloadURL != "" {
				preloadRecorder := httptest.NewRecorder()
				body := strings.NewReader(test.preloadURL)
				preloadRequest := httptest.NewRequest(http.MethodPost, "/", body)
				preloadRequest.Header.Add("Content-Type", "text/plain; charset=utf-8")
				preloadRequest.Header.Add("Content-Length", strconv.FormatInt(preloadRequest.ContentLength, 10))
				CreateShortURLHandler{}.ServeHTTP(preloadRecorder, preloadRequest)
				preloaderRes := preloadRecorder.Result()
				require.Equal(t, http.StatusCreated, preloadRecorder.Code)
				defer preloaderRes.Body.Close()
				preloaderBody, err := io.ReadAll(preloaderRes.Body)
				require.NoError(t, err)
				shortURL = strings.TrimPrefix(string(preloaderBody), "http://localhost:8080/")
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.requestMethod, "/"+shortURL, nil)
			request.SetPathValue("id", shortURL)
			RedirectToOriginalURLHandler{}.ServeHTTP(recorder, request)
			res := recorder.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			if res.StatusCode >= 400 {
				defer res.Body.Close()
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Contains(t, string(resBody), test.want.response)
			} else {
				header := res.Header.Get(test.want.header)
				assert.NotEmpty(t, header)
				assert.Equal(t, test.want.response, header)
			}
		})
	}
}
