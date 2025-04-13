package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/mocks"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"testing"
)

type RepoMock struct {
	localStorage    map[string]string
	localIDsStorage map[string][]string
}

func (rm RepoMock) Create(_ context.Context, id string, originalURL string, userID string) (string, error) {
	if rm.localStorage == nil {
		rm.localStorage = make(map[string]string)
		rm.localIDsStorage = make(map[string][]string)
	}
	rm.localStorage[id] = originalURL
	currentShortURLs := rm.localIDsStorage[userID]
	currentShortURLs = append(currentShortURLs, id)
	rm.localIDsStorage[userID] = currentShortURLs
	return id, nil
}

func (rm RepoMock) Read(_ context.Context, id string) string {
	if rm.localStorage == nil {
		rm.localStorage = make(map[string]string)
	}
	originalURL, ok := rm.localStorage[id]
	if !ok {
		return ""
	}
	return originalURL
}

func (rm RepoMock) Ping(_ context.Context) error {
	return nil
}

func (rm RepoMock) BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error) {
	results := make([]models.ShortenBatchItemResponse, 0, len(URLs))
	for shortURL, data := range URLs {
		result, err := rm.Create(ctx, shortURL, data.OriginalURL, userID)
		if err != nil {
			return nil, err
		}
		results = append(results, models.ShortenBatchItemResponse{CorrelationID: data.CorrelationID, ShortURL: result})
	}
	return results, nil
}

func (rm RepoMock) ReadByUserID(_ context.Context, userID string) ([]models.ShortURLsByUserResponse, error) {
	currentShortURLs := rm.localIDsStorage[userID]
	if len(currentShortURLs) == 0 {
		return nil, nil
	}
	result := make([]models.ShortURLsByUserResponse, len(currentShortURLs))
	for _, shortURL := range currentShortURLs {
		result = append(result, models.ShortURLsByUserResponse{
			ShortURL:    shortURL,
			OriginalURL: rm.localStorage[shortURL],
		})
	}
	return result, nil
}

func TestNewService(t *testing.T) {
	type args struct {
		repo storage.Repository
	}
	tests := []struct {
		name string
		args args
		want ShortURLService
	}{
		{
			name: "Successful creation of service",
			args: args{RepoMock{make(map[string]string), make(map[string][]string)}},
			want: ShortURLService{RepoMock{make(map[string]string), make(map[string][]string)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewService(tt.args.repo))
		})
	}
}

func TestShortURLService_Create(t *testing.T) {
	type fields struct {
		repo storage.Repository
	}
	type args struct {
		ctx         context.Context
		originalURL string
		userID      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Successful creation of short URL",
			fields:  fields{repo: RepoMock{make(map[string]string), make(map[string][]string)}},
			args:    args{originalURL: "https://ya.ru", userID: "ImagineThisIsTheUUID"},
			wantErr: false,
		},
		{
			name:   "Successful creation of short url with long original URL",
			fields: fields{repo: RepoMock{make(map[string]string), make(map[string][]string)}},
			args: args{
				ctx:         context.Background(),
				originalURL: "https://example.com/veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeerylong",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShortURLService{
				repo: tt.fields.repo,
			}
			got, err := s.Create(tt.args.ctx, tt.args.originalURL, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.IsType(t, "", got)
			assert.NotEmpty(t, got)
			assert.Contains(t, got, config.Settings.HostedOn)
		})
	}
}

func TestShortURLService_CreateWithError(t *testing.T) {
	type args struct {
		ctx         context.Context
		originalURL string
		userID      string
	}
	tests := []struct {
		name           string
		args           args
		mockReturns    string
		mockReturnsErr error
		want           string
		wantErr        bool
	}{
		{
			name:           "Creation of short URL with already existing originalURL",
			args:           args{ctx: context.Background(), originalURL: "https://ya.ru", userID: "ImagineThisIsTheUUID"},
			mockReturns:    "lelelele",
			mockReturnsErr: storage.ErrAlreadyExists,
			want:           "http://localhost:8080/lelelele",
			wantErr:        true,
		},
		{
			name:           "Creation of short URL with some other error",
			args:           args{ctx: context.Background(), originalURL: "https://ya.ru", userID: "ImagineThisIsTheUUID"},
			mockReturns:    "",
			mockReturnsErr: errors.New("some error"),
			want:           "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockRepository(ctrl)
			s := &ShortURLService{
				repo: repoMock,
			}
			repoMock.EXPECT().
				Read(tt.args.ctx, gomock.Any()).
				Return("")
			repoMock.EXPECT().
				Create(tt.args.ctx, gomock.Any(), tt.args.originalURL, tt.args.userID).
				Return(tt.mockReturns, tt.mockReturnsErr)
			got, err := s.Create(tt.args.ctx, tt.args.originalURL, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.ErrorIs(t, err, tt.mockReturnsErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShortURLService_Read(t *testing.T) {
	type fields struct {
		repo storage.Repository
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name:    "Successful read of short URL",
			fields:  fields{repo: RepoMock{map[string]string{"LElElelE": "https://ya.ru"}, map[string][]string{"ImagineThisIsTheUUID": {"LElElelE"}}}},
			args:    args{id: "LElElelE"},
			want:    "https://ya.ru",
			wantErr: nil,
		},
		{
			name:    "Unsuccessful read of short URL",
			fields:  fields{repo: RepoMock{make(map[string]string), make(map[string][]string)}},
			args:    args{id: "NoNeXiSt"},
			want:    "",
			wantErr: ErrShortURLNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShortURLService{
				repo: tt.fields.repo,
			}
			got, err := s.Read(tt.args.ctx, tt.args.id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShortURLService_FillRow(t *testing.T) {
	type fields struct {
		repo storage.Repository
	}
	type args struct {
		ctx         context.Context
		shortURL    string
		originalURL string
		userID      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Successful filling of short URL",
			fields:  fields{repo: RepoMock{make(map[string]string), make(map[string][]string)}},
			args:    args{ctx: context.Background(), originalURL: "https://ya.ru"},
			wantErr: false,
		},
		{
			name:   "Successful filling of short url with long original URL",
			fields: fields{repo: RepoMock{make(map[string]string), make(map[string][]string)}},
			args: args{
				ctx:         context.Background(),
				originalURL: "https://example.com/veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeerylong",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShortURLService{
				repo: tt.fields.repo,
			}
			err := s.FillRow(tt.args.ctx, tt.args.originalURL, tt.args.shortURL, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.fields.repo.Read(tt.args.ctx, tt.args.shortURL), tt.args.originalURL)
		})
	}
}

func Test_generateID(t *testing.T) {
	tests := []struct {
		name       string
		wantLength int
	}{
		{
			name:       "Successful generation of ID",
			wantLength: shortURLIdLength,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateID()
			assert.Equal(t, tt.wantLength, len(got))

		})
	}
}

func TestShortURLService_BatchCreate(t *testing.T) {
	type args struct {
		ctx         context.Context
		requestData []models.ShortenBatchItemRequest
		userID      string
	}
	tests := []struct {
		name    string
		args    args
		want    []models.ShortenBatchItemResponse
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Successful batch creation",
			args: args{
				ctx: context.Background(),
				requestData: []models.ShortenBatchItemRequest{
					{CorrelationID: "lele", OriginalURL: "https://ya.ru"},
					{CorrelationID: "lolo", OriginalURL: "https://yandex.ru"},
				},
				userID: "ImagineThisIsTheUUID",
			},
			want: []models.ShortenBatchItemResponse{
				{CorrelationID: "lele", ShortURL: config.Settings.HostedOn + "lelele"},
				{CorrelationID: "lolo", ShortURL: config.Settings.HostedOn + "lelele"},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Successful batch creation for single URL",
			args: args{
				ctx: context.Background(),
				requestData: []models.ShortenBatchItemRequest{
					{CorrelationID: "lele", OriginalURL: "https://ya.ru"},
				},
				userID: "ImagineThisIsTheUUID",
			},
			want: []models.ShortenBatchItemResponse{
				{CorrelationID: "lele", ShortURL: config.Settings.HostedOn + "lelele"},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockRepository(ctrl)
			s := &ShortURLService{
				repo: repoMock,
			}
			var returnStruct []models.ShortenBatchItemResponse
			for _, requestItem := range tt.args.requestData {
				returnStruct = append(returnStruct, models.ShortenBatchItemResponse{
					CorrelationID: requestItem.CorrelationID,
					ShortURL:      "lelele",
				})
			}
			repoMock.EXPECT().
				BatchCreate(tt.args.ctx, gomock.Any(), tt.args.userID).
				Return(returnStruct, nil)
			got, err := s.BatchCreate(tt.args.ctx, tt.args.requestData, tt.args.userID)
			if !tt.wantErr(t, err, fmt.Sprintf("BatchCreate(%v, %v, %v)", tt.args.ctx, tt.args.requestData, tt.args.userID)) {
				return
			}
			assert.Equalf(t, tt.want, got, "BatchCreate(%v, %v, %v)", tt.args.ctx, tt.args.requestData, tt.args.userID)
		})
	}
}

func TestShortURLService_ReadByUserID(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
	}
	tests := []struct {
		name        string
		args        args
		want        []models.ShortURLsByUserResponse
		mockReturns []models.ShortURLsByUserResponse
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name: "Successful read",
			args: args{
				ctx:    context.Background(),
				userID: "ImagineThisIsTheUUID",
			},
			mockReturns: []models.ShortURLsByUserResponse{
				{
					ShortURL:    "lelele",
					OriginalURL: "http://ya.ru",
				},
				{
					ShortURL:    "lololo",
					OriginalURL: "http://yandex.ru",
				},
			},
			want: []models.ShortURLsByUserResponse{
				{
					ShortURL:    "http://localhost:8080/lelele",
					OriginalURL: "http://ya.ru",
				},
				{
					ShortURL:    "http://localhost:8080/lololo",
					OriginalURL: "http://yandex.ru",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Successful read",
			args: args{
				ctx:    context.Background(),
				userID: "ImagineThisIsTheUUID",
			},
			mockReturns: []models.ShortURLsByUserResponse{},
			want:        []models.ShortURLsByUserResponse{},
			wantErr:     assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockRepository(ctrl)
			s := &ShortURLService{
				repo: repoMock,
			}
			repoMock.EXPECT().
				ReadByUserID(tt.args.ctx, tt.args.userID).
				Return(tt.mockReturns, nil)
			got, err := s.ReadByUserID(tt.args.ctx, tt.args.userID)
			if !tt.wantErr(t, err, fmt.Sprintf("ReadByUserID(%v, %v)", tt.args.ctx, tt.args.userID)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ReadByUserID(%v, %v)", tt.args.ctx, tt.args.userID)
		})
	}
}
