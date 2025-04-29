package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/mocks"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/service"
	"github.com/clearthree/url-shortener/internal/app/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

var ServiceForTest = service.NewService(storage.MemoryRepo{}, make(chan struct{}))

func TestNewCreateShortURLHandler(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		name string
		args args
		want *CreateShortURLHandler
	}{
		{
			name: "success",
			args: args{
				service: &ServiceForTest,
			},
			want: &CreateShortURLHandler{service: &ServiceForTest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewCreateShortURLHandler(tt.args.service), "NewCreateShortURLHandler(%v)", tt.args.service)
		})
	}
}

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
		mockReturns          string
		mockReturnsError     error
		mockExpect           bool
		want                 want
	}{
		{
			name:                 "Successful short url creation test",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "",
			mockReturns:          "http://localhost:8080/lelelele",
			mockReturnsError:     nil,
			mockExpect:           true,
			want: want{
				code:        http.StatusCreated,
				response:    `http://localhost:8080/lelelele`,
				contentType: "text/plain",
			},
		},
		{
			name:                 "Successful short url creation test",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "application/x-gzip",
			requestContentLength: "",
			mockReturns:          "http://localhost:8080/lelelele",
			mockReturnsError:     nil,
			mockExpect:           true,
			want: want{
				code:        http.StatusCreated,
				response:    `http://localhost:8080/lelelele`,
				contentType: "text/plain",
			},
		},
		{
			name:                 "Unsuccessful request due to wrong content-type",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "application/json",
			requestContentLength: "",
			mockReturns:          "",
			mockReturnsError:     nil,
			mockExpect:           false,
			want: want{
				code:        http.StatusBadRequest,
				response:    `Only text/plain or application/x-gzip content types are allowed`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                 "Unsuccessful request due to wrong content-length",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "писятДва",
			mockReturns:          "",
			mockReturnsError:     nil,
			mockExpect:           false,
			want: want{
				code:        http.StatusBadRequest,
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
			mockReturns:          "",
			mockReturnsError:     nil,
			mockExpect:           false,
			want: want{
				code:        http.StatusBadRequest,
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
			mockExpect:           false,
			want: want{
				code:        http.StatusBadRequest,
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
			mockReturns:          "",
			mockReturnsError:     nil,
			mockExpect:           false,
			want: want{
				code:        http.StatusBadRequest,
				response:    `The provided payload is not a valid URL`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                 "Successful short url return with conflict",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "",
			mockReturns:          "http://localhost:8080/lelelele",
			mockReturnsError:     storage.ErrAlreadyExists,
			mockExpect:           true,
			want: want{
				code:        http.StatusConflict,
				response:    `http://localhost:8080/lelelele`,
				contentType: "text/plain",
			},
		},
		{
			name:                 "Successful short url return with conflict",
			requestPayload:       "https://ya.ru",
			requestMethod:        http.MethodPost,
			requestContentType:   "text/plain",
			requestContentLength: "",
			mockReturns:          "http://localhost:8080/lelelele",
			mockReturnsError:     errors.New("blablabla"),
			mockExpect:           true,
			want: want{
				code:        http.StatusBadRequest,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			if test.mockExpect {
				shortURLServiceMock.EXPECT().
					Create(context.Background(), gomock.Any(), gomock.Any()).
					Return(test.mockReturns, test.mockReturnsError)
			}

			body := strings.NewReader(test.requestPayload)
			request := httptest.NewRequest(test.requestMethod, "/", body)
			request.Header.Add("Content-Type", test.requestContentType)
			if test.requestContentLength != "" {
				request.Header.Add("Content-Length", test.requestContentLength)
			} else {
				request.Header.Add("Content-Length", strconv.FormatInt(request.ContentLength, 10))
			}
			recorder := httptest.NewRecorder()
			CreateShortURLHandler{shortURLServiceMock}.ServeHTTP(recorder, request)

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

func TestNewRedirectToOriginalURLHandler(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		name string
		args args
		want *RedirectToOriginalURLHandler
	}{
		{
			name: "success",
			args: args{
				service: &ServiceForTest,
			},
			want: &RedirectToOriginalURLHandler{service: &ServiceForTest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewRedirectToOriginalURLHandler(tt.args.service), "NewRedirectToOriginalURLHandler(%v)", tt.args.service)
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
		requestMethod string
		mockValue     string
		want          want
	}{
		{
			name:          "Successful redirection test",
			requestMethod: http.MethodGet,
			mockValue:     "lelelele",
			want: want{
				code:     http.StatusTemporaryRedirect,
				response: "https://ya.ru",
				header:   "Location",
			},
		},
		{
			name:          "Unsuccessful redirection due to non-existing shortURL test",
			requestMethod: http.MethodGet,
			mockValue:     "",
			want: want{
				code:     http.StatusNotFound,
				response: "Short url not found",
				header:   "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var shortURL = "lelelele"
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			if test.mockValue != "" {
				shortURLServiceMock.EXPECT().Read(context.Background(), test.mockValue).Return(test.want.response, false, nil)
			} else {
				shortURLServiceMock.EXPECT().Read(context.Background(), gomock.Any()).Return("", false, service.ErrShortURLNotFound)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.requestMethod, "/"+shortURL, nil)
			request.SetPathValue("id", shortURL)
			handler := NewRedirectToOriginalURLHandler(shortURLServiceMock)
			handler.ServeHTTP(recorder, request)
			res := recorder.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			if res.StatusCode >= http.StatusBadRequest {
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

func TestNewCreateJSONShortURLHandler(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		name string
		args args
		want *CreateJSONShortURLHandler
	}{
		{
			name: "success",
			args: args{
				service: &ServiceForTest,
			},
			want: &CreateJSONShortURLHandler{service: &ServiceForTest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewCreateJSONShortURLHandler(tt.args.service), "NewCreateJSONShortURLHandler(%v)", tt.args.service)
		})
	}
}

func TestCreateJSONShortURLHandler_ServeHTTP(t *testing.T) {
	type want struct {
		code        int
		contentType string
		errMessage  string
	}
	tests := []struct {
		name               string
		requestPayload     string
		requestContentType string
		mockExpect         bool
		want               want
	}{
		{
			name:               "Successful creation of the short URL",
			requestPayload:     `{"url": "https://ya.ru"}`,
			requestContentType: "application/json",
			mockExpect:         true,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				errMessage:  "",
			},
		},
		{
			name:               "Empty URL passed",
			requestPayload:     `{"url": ""}`,
			requestContentType: "application/json",
			mockExpect:         false,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
				errMessage:  "Please provide an url\n",
			},
		},
		{
			name:               "Invalid URL passed",
			requestPayload:     `{"url": "asdasdsa"}`,
			requestContentType: "application/json",
			mockExpect:         false,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
				errMessage:  "The provided payload is not a valid URL\n",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			if test.mockExpect {
				shortURLServiceMock.EXPECT().Create(context.Background(), gomock.Any(), gomock.Any()).Return("http://localhost:8080/lelelele", nil)
			}
			body := strings.NewReader(test.requestPayload)
			request := httptest.NewRequest(http.MethodPost, "/", body)
			request.Header.Add("Content-Type", test.requestContentType)
			recorder := httptest.NewRecorder()
			handler := NewCreateJSONShortURLHandler(shortURLServiceMock)
			handler.ServeHTTP(recorder, request)
			res := recorder.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			defer res.Body.Close()
			if test.want.errMessage != "" {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, test.want.errMessage, string(resBody))
				return
			}
			var responseData models.ShortenResponse
			dec := json.NewDecoder(res.Body)
			err := dec.Decode(&responseData)
			require.NoError(t, err)
			assert.Contains(t, responseData.Result, config.Settings.HostedOn)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestNewPingHandler(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		name string
		args args
		want *PingHandler
	}{
		{
			name: "success",
			args: args{
				service: &ServiceForTest,
			},
			want: &PingHandler{service: &ServiceForTest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewPingHandler(tt.args.service), "NewPingHandler(%v)", tt.args.service)
		})
	}
}

func TestPingHandler_ServeHTTP(t *testing.T) {
	type want struct {
		code       int
		errMessage string
	}
	tests := []struct {
		name        string
		wantSuccess bool
		want        want
	}{
		{
			name:        "Successful ping",
			wantSuccess: true,
			want: want{
				code:       200,
				errMessage: "",
			},
		},
		{
			name:        "Unsuccessful ping",
			wantSuccess: false,
			want: want{
				code:       500,
				errMessage: "Database is not available\n",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			if test.wantSuccess {
				shortURLServiceMock.EXPECT().Ping(context.Background()).Return(nil)
			} else {
				shortURLServiceMock.EXPECT().Ping(context.Background()).Return(errors.New("database is not available"))
			}
			request := httptest.NewRequest(http.MethodGet, "/ping", nil)
			recorder := httptest.NewRecorder()
			pingHandler := NewPingHandler(shortURLServiceMock)
			pingHandler.ServeHTTP(recorder, request)
			res := recorder.Result()
			defer res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			if res.StatusCode > http.StatusOK {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, test.want.errMessage, string(resBody))
			}
		})
	}
}

func TestNewBatchCreateShortURLHandler(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		name string
		args args
		want *BatchCreateShortURLHandler
	}{
		{
			name: "success",
			args: args{
				service: &ServiceForTest,
			},
			want: &BatchCreateShortURLHandler{
				service: &ServiceForTest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewBatchCreateShortURLHandler(tt.args.service), "NewBatchCreateShortURLHandler(%v)", tt.args.service)
		})
	}
}

func TestBatchCreateShortURLHandler_ServeHTTP(t *testing.T) {
	type want struct {
		code        int
		contentType string
		payload     []models.ShortenBatchItemResponse
		errMessage  string
	}
	tests := []struct {
		name               string
		requestPayload     string
		requestContentType string
		mockExpect         bool
		want               want
	}{
		{
			name: "Successful creation of batch for short url",
			requestPayload: `[
				{"original_url": "https://ya.ru", "correlation_id": "lelele"},
				{"original_url": "https://yandex.ru", "correlation_id": "lololo"}
			]`,
			requestContentType: "application/json",
			mockExpect:         true,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				payload: []models.ShortenBatchItemResponse{
					{CorrelationID: "lelele", ShortURL: "http://localhost:8080/LELELELE"},
					{CorrelationID: "lololo", ShortURL: "http://localhost:8080/LELELELE"},
				},
				errMessage: "",
			},
		},
		{
			name:               "Successful creation of batch for short url with single url",
			requestPayload:     `[{"original_url": "https://ya.ru", "correlation_id": "lelele"}]`,
			requestContentType: "application/json",
			mockExpect:         true,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				payload: []models.ShortenBatchItemResponse{
					{CorrelationID: "lelele", ShortURL: "http://localhost:8080/LELELELE"},
				},
				errMessage: "",
			},
		},
		{
			name:               "Unsuccessful creation of batch for short url with empty batch",
			requestPayload:     `[]`,
			requestContentType: "application/json",
			mockExpect:         false,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
				payload:     nil,
				errMessage:  "Please provide a batch of URLs\n",
			},
		},
		{
			name: "Successful creation of batch for short url",
			requestPayload: `[
				{"original_url": "lele://ya.ru", "correlation_id": "lelele"},
				{"original_url": "https://yandex.ru", "correlation_id": "lololo"}
			]`,
			requestContentType: "application/json",
			mockExpect:         false,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
				payload:     nil,
				errMessage:  "One of the provided items is not a valid URL\n",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			if test.mockExpect {
				var requestData []models.ShortenBatchItemRequest
				dec := json.NewDecoder(strings.NewReader(test.requestPayload))
				if err := dec.Decode(&requestData); err != nil {
					require.NoError(t, err, "failed to decode test.requestPayload")
				}
				var returnStruct []models.ShortenBatchItemResponse
				for _, requestItem := range requestData {
					returnStruct = append(returnStruct, models.ShortenBatchItemResponse{
						CorrelationID: requestItem.CorrelationID,
						ShortURL:      "http://localhost:8080/LELELELE",
					})
				}
				shortURLServiceMock.EXPECT().
					BatchCreate(context.Background(), requestData, gomock.Any()).
					Return(returnStruct, nil)
			}
			body := strings.NewReader(test.requestPayload)
			request := httptest.NewRequest(http.MethodPost, "/", body)
			request.Header.Add("Content-Type", test.requestContentType)
			recorder := httptest.NewRecorder()
			handler := NewBatchCreateShortURLHandler(shortURLServiceMock)
			handler.ServeHTTP(recorder, request)
			res := recorder.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			defer res.Body.Close()
			if test.want.errMessage != "" {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, test.want.errMessage, string(resBody))
				return
			}
			var responseData []models.ShortenBatchItemResponse
			dec := json.NewDecoder(res.Body)
			err := dec.Decode(&responseData)
			require.NoError(t, err)
			assert.Equal(t, test.want.payload, responseData)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestNewGetAllURLsForUserHandler(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		name string
		args args
		want *GetAllURLsForUserHandler
	}{
		{
			name: "Successful creation of all URLs for user handler",
			args: args{
				service: &ServiceForTest,
			},
			want: &GetAllURLsForUserHandler{
				service: &ServiceForTest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewGetAllURLsForUserHandler(tt.args.service), "NewGetAllURLsForUserHandler(%v)", tt.args.service)
		})
	}
}

func TestGetAllURLsForUserHandler_ServeHTTP(t *testing.T) {
	type want struct {
		code        int
		contentType string
		payload     []models.ShortURLsByUserResponse
	}
	tests := []struct {
		name      string
		want      want
		mockValue []models.ShortURLsByUserResponse
	}{
		{
			name: "Successful return of all URLs for the user",
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
				payload: []models.ShortURLsByUserResponse{
					{
						ShortURL:    "http://localhost:8080/lelele",
						OriginalURL: "http://ya.ru",
					},
					{
						ShortURL:    "http://localhost:8080/lololo",
						OriginalURL: "http://yandex.ru",
					},
				},
			},
			mockValue: []models.ShortURLsByUserResponse{
				{
					ShortURL:    "http://localhost:8080/lelele",
					OriginalURL: "http://ya.ru",
				},
				{
					ShortURL:    "http://localhost:8080/lololo",
					OriginalURL: "http://yandex.ru",
				},
			},
		},
		{
			name: "Successful return of empty URLs list",
			want: want{
				code:        http.StatusNoContent,
				contentType: "application/json",
				payload:     nil,
			},
			mockValue: []models.ShortURLsByUserResponse{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			shortURLServiceMock.EXPECT().
				ReadByUserID(context.Background(), gomock.Any()).
				Return(test.mockValue, nil)
			request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
			recorder := httptest.NewRecorder()
			handler := NewGetAllURLsForUserHandler(shortURLServiceMock)
			handler.ServeHTTP(recorder, request)
			res := recorder.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			defer res.Body.Close()

			if test.want.payload != nil {
				var responseData []models.ShortURLsByUserResponse
				dec := json.NewDecoder(res.Body)
				err := dec.Decode(&responseData)
				require.NoError(t, err)
				assert.Equal(t, test.want.payload, responseData)
				assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}
