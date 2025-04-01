package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDBRepo_Create(t *testing.T) {
	type args struct {
		ctx         context.Context
		id          string
		originalURL string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				ctx:         context.Background(),
				id:          "lelelele",
				originalURL: "http://ya.ru",
			},
			want:    "lelelele",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)

			D := DBRepo{
				pool: db,
			}
			mock.ExpectPrepare("INSERT INTO short_url").ExpectExec().
				WithArgs(tt.args.id, tt.args.originalURL).
				WillReturnResult(sqlmock.NewResult(1, 1))
			got, err := D.Create(tt.args.ctx, tt.args.id, tt.args.originalURL)
			if !tt.wantErr(t, err, fmt.Sprintf("Create(%v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.originalURL)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Create(%v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.originalURL)
		})
	}
}

func TestDBRepo_Ping(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		args      args
		wantErr   assert.ErrorAssertionFunc
		returnErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
			},
			wantErr:   assert.NoError,
			returnErr: false,
		},
		{
			name: "error",
			args: args{
				ctx: context.Background(),
			},
			wantErr:   assert.Error,
			returnErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			D := DBRepo{
				pool: db,
			}
			if tt.returnErr {
				mock.ExpectPing().WillReturnError(errors.New("error"))
			} else {
				mock.ExpectPing()
			}
			tt.wantErr(t, D.Ping(tt.args.ctx), fmt.Sprintf("Ping(%v)", tt.args.ctx))
		})
	}
}

func TestDBRepo_Read(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				id:  "lelelele",
			},
			want: "lelelele",
		},
		{
			name: "not found",
			args: args{
				ctx: context.Background(),
				id:  "lelelele",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			D := DBRepo{
				pool: db,
			}
			mock.ExpectPrepare("SELECT original_url FROM short_url").ExpectQuery().
				WithArgs(tt.args.id).
				WillReturnRows(mock.NewRows([]string{"original_url"}).AddRow(tt.want))
			assert.Equalf(t, tt.want, D.Read(tt.args.ctx, tt.args.id), "Read(%v, %v)", tt.args.ctx, tt.args.id)
		})
	}
}

func TestNewDBRepo(t *testing.T) {
	type args struct {
		pool *sql.DB
	}
	tests := []struct {
		name string
		args args
		want *DBRepo
	}{
		{
			name: "success",
			args: args{
				pool: nil,
			},
			want: &DBRepo{
				pool: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewDBRepo(tt.args.pool), "NewDBRepo(%v)", tt.args.pool)
		})
	}
}

func TestDBRepo_BatchCreate(t *testing.T) {
	type args struct {
		ctx  context.Context
		URLs map[string]models.ShortenBatchItemRequest
	}
	tests := []struct {
		name    string
		args    args
		want    []models.ShortenBatchItemResponse
		wantErr assert.ErrorAssertionFunc
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
			wantErr: assert.NoError,
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
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			D := DBRepo{
				pool: db,
			}
			mock.ExpectBegin()
			mockStatement := mock.ExpectPrepare("INSERT INTO short_url")
			for range tt.args.URLs {
				mockStatement.
					ExpectExec().
					WillReturnResult(sqlmock.NewResult(1, 1))
			}
			mock.ExpectCommit()
			got, err := D.BatchCreate(tt.args.ctx, tt.args.URLs)
			if !tt.wantErr(t, err, fmt.Sprintf("BatchCreate(%v, %v)", tt.args.ctx, tt.args.URLs)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "BatchCreate(%v, %v)", tt.args.ctx, tt.args.URLs)
		})
	}
}
