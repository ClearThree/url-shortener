package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/mocks"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/storage"

	"testing"
)

type RepoMock struct {
	localStorage                map[string]string
	localIDsStorage             map[string][]string
	localStorageUsersByURLs     map[string]string
	localStorageDeactivatedURLs map[string]bool
}

func (rm RepoMock) Create(_ context.Context, id string, originalURL string, userID string) (string, error) {
	if rm.localStorage == nil {
		rm.localStorage = make(map[string]string)
		rm.localIDsStorage = make(map[string][]string)
		rm.localStorageUsersByURLs = make(map[string]string)
	}
	rm.localStorage[id] = originalURL
	rm.localStorageUsersByURLs[id] = userID
	currentShortURLs := rm.localIDsStorage[userID]
	currentShortURLs = append(currentShortURLs, id)
	rm.localIDsStorage[userID] = currentShortURLs
	return id, nil
}

func (rm RepoMock) Read(_ context.Context, id string) (string, bool) {
	if rm.localStorage == nil {
		rm.localStorage = make(map[string]string)
		rm.localStorageDeactivatedURLs = make(map[string]bool)
	}
	originalURL, ok := rm.localStorage[id]
	if !ok {
		return "", false
	}
	_, deleted := rm.localStorageDeactivatedURLs[id]
	return originalURL, deleted
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
		_, deleted := rm.localStorageDeactivatedURLs[shortURL]
		if deleted {
			continue
		}
		result = append(result, models.ShortURLsByUserResponse{
			ShortURL:    shortURL,
			OriginalURL: rm.localStorage[shortURL],
		})
	}
	return result, nil
}

func (rm RepoMock) GetUserIDByShortURL(_ context.Context, shortURL string) (string, error) {
	_, ok := rm.localStorageDeactivatedURLs[shortURL]
	if ok {
		return "", nil
	}
	return rm.localStorageUsersByURLs[shortURL], nil
}

func (rm RepoMock) SetURLsInactive(_ context.Context, shortURLs []string) error {
	for _, shortURL := range shortURLs {
		rm.localStorageDeactivatedURLs[shortURL] = true
	}
	return nil
}

func (rm RepoMock) GetStats(_ context.Context) (models.ServiceStats, error) {
	response := models.ServiceStats{
		Users: len(rm.localIDsStorage),
		URLs:  len(rm.localStorage),
	}
	return response, nil
}

func TestNewService(t *testing.T) {
	type args struct {
		repo     storage.Repository
		doneChan chan struct{}
	}
	tests := []struct {
		want ShortURLService
		args args
		name string
	}{
		{
			name: "Successful creation of service",
			args: args{RepoMock{
				make(map[string]string),
				make(map[string][]string),
				make(map[string]string),
				make(map[string]bool),
			},
				make(chan struct{})},
			want: ShortURLService{
				repo: RepoMock{
					make(map[string]string),
					make(map[string][]string),
					make(map[string]string),
					make(map[string]bool),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assert.Equal(t, tt.want.repo, NewService(tt.args.repo, tt.args.doneChan).repo)
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
			name: "Successful creation of short URL",
			fields: fields{repo: RepoMock{
				make(map[string]string),
				make(map[string][]string),
				make(map[string]string),
				make(map[string]bool),
			}},
			args:    args{originalURL: "https://ya.ru", userID: "ImagineThisIsTheUUID"},
			wantErr: false,
		},
		{
			name: "Successful creation of short url with long original URL",
			fields: fields{repo: RepoMock{
				make(map[string]string),
				make(map[string][]string),
				make(map[string]string),
				make(map[string]bool),
			}},
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
				Return("", false)
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
		fields  fields
		wantErr error
		args    args
		name    string
		want    string
	}{
		{
			name: "Successful read of short URL",
			fields: fields{repo: RepoMock{
				map[string]string{"LElElelE": "https://ya.ru"},
				map[string][]string{"ImagineThisIsTheUUID": {"LElElelE"}},
				map[string]string{"LElElelE": "ImagineThisIsTheUUID"},
				map[string]bool{}},
			},
			args:    args{id: "LElElelE"},
			want:    "https://ya.ru",
			wantErr: nil,
		},
		{
			name: "Unsuccessful read of short URL",
			fields: fields{
				repo: RepoMock{
					make(map[string]string),
					make(map[string][]string),
					map[string]string{"LElElelE": "ImagineThisIsTheUUID"},
					map[string]bool{},
				},
			},
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
			got, deleted, err := s.Read(tt.args.ctx, tt.args.id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, false, deleted)
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
			name: "Successful filling of short URL",
			fields: fields{repo: RepoMock{
				make(map[string]string),
				make(map[string][]string),
				make(map[string]string),
				make(map[string]bool),
			}},
			args:    args{ctx: context.Background(), originalURL: "https://ya.ru"},
			wantErr: false,
		},
		{
			name: "Successful filling of short url with long original URL",
			fields: fields{repo: RepoMock{
				make(map[string]string),
				make(map[string][]string),
				make(map[string]string),
				make(map[string]bool),
			}},
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
			res, deleted := tt.fields.repo.Read(tt.args.ctx, tt.args.shortURL)
			assert.Equal(t, tt.args.originalURL, res)
			assert.Equal(t, false, deleted)
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
		userID      string
		requestData []models.ShortenBatchItemRequest
	}
	tests := []struct {
		wantErr assert.ErrorAssertionFunc
		args    args
		name    string
		want    []models.ShortenBatchItemResponse
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
		wantErr     assert.ErrorAssertionFunc
		args        args
		name        string
		want        []models.ShortURLsByUserResponse
		mockReturns []models.ShortURLsByUserResponse
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

func BenchmarkShortURLService(b *testing.B) {
	repo := RepoMock{
		make(map[string]string),
		make(map[string][]string),
		make(map[string]string),
		make(map[string]bool),
	}
	service := NewService(repo, make(chan struct{}))
	ctx := context.Background()
	testCaseLength := 10
	testUserID := "ImagineThisIsTheUUID"
	URLs := make([]string, testCaseLength)
	for i := 0; i < testCaseLength; i++ {
		URLs[i] = "http://yandex" + strconv.Itoa(i) + ".ru"
	}
	shortURL, err := service.Create(ctx, URLs[0], testUserID)
	if err != nil {
		panic(err)
	}
	testID := strings.Split(shortURL, "/")[3]

	b.ResetTimer()
	b.Run("ReadByUserID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, innerErr := service.ReadByUserID(ctx, testUserID)
			if innerErr != nil {
				panic(innerErr)
			}
		}
	})
	b.Run("Create", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = service.Create(ctx, "http://ya.ru", testUserID)
			if err != nil {
				panic(err)
			}
		}
	})
	b.Run("Read", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err = service.Read(ctx, testID)
			if err != nil {
				panic(err)
			}
		}
	})
	b.Run("BatchCreate", func(b *testing.B) {
		b.StopTimer()
		requestData := make([]models.ShortenBatchItemRequest, testCaseLength)
		for i := 1; i < testCaseLength; i++ {
			requestData[i] = models.ShortenBatchItemRequest{
				OriginalURL:   URLs[i],
				CorrelationID: strconv.Itoa(i),
			}
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			_, err = service.BatchCreate(ctx, requestData, testUserID)
			if err != nil {
				panic(err)
			}
		}
	})
}

func TestShortURLService_GetStats(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		args    args
		name    string
		want    models.ServiceStats
		wantErr bool
	}{
		{
			name: "Successful read",
			args: args{
				ctx: context.Background(),
			},
			want: models.ServiceStats{
				Users: 1337,
				URLs:  1338,
			},
			wantErr: false,
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
				GetStats(tt.args.ctx).
				Return(tt.want, nil)
			got, err := s.GetStats(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
