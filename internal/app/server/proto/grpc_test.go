package proto

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/clearthree/url-shortener/internal/app/mocks"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/service"
	"github.com/clearthree/url-shortener/internal/app/storage"
)

var ServiceForTest = service.NewService(storage.MemoryRepo{}, make(chan struct{}))

func TestNewShortenerGRPCServer(t *testing.T) {
	type args struct {
		service service.ShortURLServiceInterface
	}
	tests := []struct {
		args args
		want *ShortenerGRPCServer
		name string
	}{
		{
			name: "NewShortenerGRPCServer",
			args: args{
				service: &ServiceForTest,
			},
			want: &ShortenerGRPCServer{
				service: &ServiceForTest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewShortenerGRPCServer(tt.args.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewShortenerGRPCServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortenerGRPCServer_BatchCreateShortURL(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *BatchShortenRequest
	}
	tests := []struct {
		name      string
		args      args
		mockValue []models.ShortenBatchItemResponse
		wantErr   bool
	}{
		{
			name: "BatchCreateShortURL empty URLs",
			args: args{
				ctx: context.Background(),
				request: &BatchShortenRequest{
					Items: make([]*BatchShortenRequest_Item, 0),
				},
			},
			wantErr: true,
		},
		{
			name: "BatchCreateShortURL empty UserID",
			args: args{
				ctx: context.Background(),
				request: &BatchShortenRequest{
					Items: []*BatchShortenRequest_Item{
						{
							OriginalUrl:   "http://ya.ru",
							CorrelationId: "s",
						},
						{
							OriginalUrl:   "http://ya.ru",
							CorrelationId: "s",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "BatchCreateShortURL success",
			args: args{
				ctx: context.Background(),
				request: &BatchShortenRequest{
					Items: []*BatchShortenRequest_Item{
						{
							OriginalUrl:   "http://ya.ru",
							CorrelationId: "s",
						},
						{
							OriginalUrl:   "http://ya.ru",
							CorrelationId: "s",
						},
					},
					UserId: "lele",
				},
			},
			wantErr: false,
			mockValue: []models.ShortenBatchItemResponse{
				{CorrelationID: "lelele", ShortURL: "http://localhost:8080/LELELELE"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			requestData := make([]models.ShortenBatchItemRequest, len(tt.args.request.Items))
			for i, item := range tt.args.request.Items {
				requestData[i] = models.ShortenBatchItemRequest{
					CorrelationID: item.CorrelationId,
					OriginalURL:   item.OriginalUrl,
				}
			}
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			if !tt.wantErr {
				shortURLServiceMock.EXPECT().
					BatchCreate(context.Background(), requestData, tt.args.request.UserId).
					Return(tt.mockValue, nil)
			}
			s := NewShortenerGRPCServer(shortURLServiceMock)
			response, err := s.BatchCreateShortURL(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("BatchCreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, len(tt.mockValue), len(response.Items))
			}
		})
	}
}

func TestShortenerGRPCServer_CreateShortURL(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *ShortenRequest
	}
	tests := []struct {
		name      string
		args      args
		mockValue string
		wantErr   bool
	}{
		{
			name: "CreateShortURL empty URL",
			args: args{
				ctx: context.Background(),
				request: &ShortenRequest{
					UserId: "lele",
				},
			},
			wantErr: true,
		},
		{
			name: "CreateShortURL empty UserID",
			args: args{
				ctx: context.Background(),
				request: &ShortenRequest{
					Url: "http://ya.ru",
				},
			},
			wantErr: true,
		},
		{
			name: "CreateShortURL success",
			args: args{
				ctx: context.Background(),
				request: &ShortenRequest{
					Url:    "http://ya.ru",
					UserId: "lele",
				},
			},
			wantErr:   false,
			mockValue: "http://localhost:8080/LELELELE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			s := NewShortenerGRPCServer(shortURLServiceMock)
			if !tt.wantErr {
				shortURLServiceMock.EXPECT().
					Create(context.Background(), tt.args.request.Url, tt.args.request.UserId).
					Return(tt.mockValue, nil)
			}
			got, err := s.CreateShortURL(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, got.Result, tt.mockValue)
			}
		})
	}
}

func TestShortenerGRPCServer_DeleteBatchURLs(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *DeleteBatchRequest
	}
	tests := []struct {
		args    args
		name    string
		wantErr bool
	}{
		{
			name: "DeleteBatchURLs empty URLs",
			args: args{
				ctx: context.Background(),
				request: &DeleteBatchRequest{
					UserId: "lele",
				},
			},
			wantErr: true,
		},
		{
			name: "DeleteBatchURLs empty UserId",
			args: args{
				ctx: context.Background(),
				request: &DeleteBatchRequest{
					ShortUrls: []string{"http://ya.ru", "http://ya2.ru"},
				},
			},
			wantErr: true,
		},
		{
			name: "DeleteBatchURLs success",
			args: args{
				ctx: context.Background(),
				request: &DeleteBatchRequest{
					ShortUrls: []string{"http://ya.ru", "http://ya2.ru"},
					UserId:    "lele",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			s := NewShortenerGRPCServer(shortURLServiceMock)
			if !tt.wantErr {
				requestPrepared := make([]models.ShortURLChannelMessage, len(tt.args.request.ShortUrls))
				for i, requestItem := range tt.args.request.ShortUrls {
					requestPrepared[i] = models.ShortURLChannelMessage{
						Ctx:      context.Background(),
						ShortURL: requestItem,
						UserID:   tt.args.request.UserId,
					}
				}
				shortURLServiceMock.EXPECT().
					ScheduleDeletionOfBatch(requestPrepared)
			}
			_, err := s.DeleteBatchURLs(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteBatchURLs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestShortenerGRPCServer_GetServiceStats(t *testing.T) {
	type args struct {
		ctx context.Context
		in1 *ServiceStatsRequest
	}
	tests := []struct {
		args    args
		want    *models.ServiceStats
		name    string
		wantErr bool
	}{
		{
			name: "GetServiceStats success",
			args: args{
				ctx: context.Background(),
			},
			want: &models.ServiceStats{
				Users: 13,
				URLs:  37,
			},
			wantErr: false,
		},
		{
			name: "GetServiceStats error",
			args: args{
				ctx: context.Background(),
			},
			want:    &models.ServiceStats{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			s := NewShortenerGRPCServer(shortURLServiceMock)
			if !tt.wantErr {
				shortURLServiceMock.EXPECT().
					GetStats(context.Background()).Return(tt.want, nil)
			} else {
				shortURLServiceMock.EXPECT().
					GetStats(context.Background()).Return(&models.ServiceStats{}, errors.New("service error"))
			}
			got, err := s.GetServiceStats(tt.args.ctx, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, got.Users, uint32(tt.want.Users), "GetServiceStats() Users got = %v, want %v", got, tt.want)
				assert.Equal(t, got.Urls, uint32(tt.want.URLs), "GetServiceStats() URLs got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortenerGRPCServer_GetUserURLs(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *GetUserURLsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    []models.ShortURLsByUserResponse
		wantErr bool
	}{
		{
			name: "GetUserURLs success",
			args: args{
				ctx: context.Background(),
				request: &GetUserURLsRequest{
					UserId: "lele",
				},
			},
			want: []models.ShortURLsByUserResponse{
				{
					ShortURL:    "http://localhost:8080/lele",
					OriginalURL: "http://ya.ru",
				},
				{
					ShortURL:    "http://localhost:8080/lelele",
					OriginalURL: "http://yax.ru",
				},
			},
			wantErr: false,
		},
		{
			name: "GetUserURLs error",
			args: args{
				ctx: context.Background(),
				request: &GetUserURLsRequest{
					UserId: "lele",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			s := NewShortenerGRPCServer(shortURLServiceMock)
			if !tt.wantErr {
				shortURLServiceMock.EXPECT().
					ReadByUserID(context.Background(), tt.args.request.UserId).Return(tt.want, nil)
			} else {
				shortURLServiceMock.EXPECT().
					ReadByUserID(context.Background(), tt.args.request.UserId).Return(nil, errors.New("service error"))
			}
			got, err := s.GetUserURLs(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserURLs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, len(got.Urls), len(tt.want), "GetUserURLs() len mismatch got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortenerGRPCServer_Ping(t *testing.T) {
	type args struct {
		ctx context.Context
		in1 *emptypb.Empty
	}
	tests := []struct {
		args    args
		want    *emptypb.Empty
		name    string
		wantErr bool
	}{
		{
			name: "Ping success",
			args: args{
				ctx: context.Background(),
				in1: &emptypb.Empty{},
			},
			want:    &emptypb.Empty{},
			wantErr: false,
		},
		{
			name: "Ping failure",
			args: args{
				ctx: context.Background(),
				in1: &emptypb.Empty{},
			},
			want:    &emptypb.Empty{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			shortURLServiceMock := mocks.NewMockShortURLServiceInterface(ctrl)
			s := NewShortenerGRPCServer(shortURLServiceMock)
			if !tt.wantErr {
				shortURLServiceMock.EXPECT().
					Ping(context.Background()).Return(nil)
			} else {
				shortURLServiceMock.EXPECT().
					Ping(context.Background()).Return(errors.New("service error"))
			}
			got, err := s.Ping(tt.args.ctx, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Ping() got = %v, want %v", got, tt.want)
			}
		})
	}
}
