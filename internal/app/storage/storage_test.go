package storage

import (
	"context"
	"fmt"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestMemoryRepo_Create(t *testing.T) {
	type args struct {
		ctx         context.Context
		id          string
		originalURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Successful addition of URL to memory",
			args: args{context.Background(), "lele", "https://ya.ru"},
			want: "lele",
		},
		{
			// The Repository doesn't even have to know what kind of data it stores, so let's check it out
			name: "Successful addition of something to memory",
			args: args{context.Background(), "lele", "something"},
			want: "lele",
		},
		{
			// It also doesn't care about any business logic limitations for keys, values etc.
			name: "Successful addition of something with long ID to memory",
			args: args{context.Background(), "longerThanUsualID", "definitelyNotAnURL"},
			want: "longerThanUsualID",
		},
		{
			// Empty key is also not a problem, even though it's an impossible case
			name: "Empty key success",
			args: args{context.Background(), "", "doesntMatter"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MemoryRepo{}
			if got, err := m.Create(tt.args.ctx, tt.args.id, tt.args.originalURL); got != tt.want {
				require.NoError(t, err)
				t.Errorf("Create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryRepo_Read(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		args    args
		preLoad map[string]string
		want    string
	}{
		{
			name: "Successful read",
			args: args{context.Background(), "lele"},
			preLoad: map[string]string{
				"lele": "https://ya.ru", "lolo": "https://ya.ru", "hehe": "https://vk.com",
			},
			want: "https://ya.ru",
		},
		{
			name:    "Unsuccessful read",
			args:    args{context.Background(), "nonExistent"},
			preLoad: map[string]string{},
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MemoryRepo{}
			for k, v := range tt.preLoad {
				_, err := m.Create(context.Background(), k, v)
				require.NoError(t, err)
			}
			got := m.Read(tt.args.ctx, tt.args.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemoryRepo_Ping(t *testing.T) {
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Successful ping",
			args: args{
				context.Background(),
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MemoryRepo{}
			tt.wantErr(t, m.Ping(tt.args.in0), fmt.Sprintf("Ping(%v)", tt.args.in0))
		})
	}
}

func TestMemoryRepo_BatchCreate(t *testing.T) {
	type args struct {
		ctx  context.Context
		URLs map[string]models.ShortenBatchItemRequest
	}
	tests := []struct {
		name string
		args args
		want []models.ShortenBatchItemResponse
	}{
		{
			name: "Successful batch create",
			args: args{
				ctx: context.Background(),
				URLs: map[string]models.ShortenBatchItemRequest{
					"lele": {CorrelationID: "lelele", OriginalURL: "https://ya.ru"},
					"lolo": {CorrelationID: "lololo", OriginalURL: "https://yandex.ru"},
				},
			},
			want: []models.ShortenBatchItemResponse{
				{CorrelationID: "lelele", ShortURL: "lele"},
				{CorrelationID: "lololo", ShortURL: "lolo"},
			},
		},
		{
			name: "Successful batch create for single URL",
			args: args{
				ctx: context.Background(),
				URLs: map[string]models.ShortenBatchItemRequest{
					"lele": {CorrelationID: "lelele", OriginalURL: "https://ya.ru"},
				},
			},
			want: []models.ShortenBatchItemResponse{
				{CorrelationID: "lelele", ShortURL: "lele"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MemoryRepo{}
			got, err := m.BatchCreate(tt.args.ctx, tt.args.URLs)
			require.NoError(t, err)
			eq := reflect.DeepEqual(got, tt.want)
			assert.Equal(t, eq, true)
		})
	}
}
