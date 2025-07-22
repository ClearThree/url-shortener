package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clearthree/url-shortener/internal/app/models"
)

func TestDBRepo_Create(t *testing.T) {
	type args struct {
		ctx         context.Context
		id          string
		originalURL string
		userID      string
	}
	tests := []struct {
		wantErr assert.ErrorAssertionFunc
		args    args
		name    string
		want    string
	}{
		{
			name: "success",
			args: args{
				ctx:         context.Background(),
				id:          "lelelele",
				originalURL: "http://ya.ru",
				userID:      "SomeUserID",
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
			mock.ExpectBegin()
			mock.ExpectPrepare("INSERT INTO users").ExpectExec().
				WithArgs(tt.args.userID).
				WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectPrepare("INSERT INTO short_url").ExpectExec().
				WithArgs(tt.args.id, tt.args.originalURL, tt.args.userID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			got, err := D.Create(tt.args.ctx, tt.args.id, tt.args.originalURL, tt.args.userID)
			if !tt.wantErr(t, err, fmt.Sprintf("Create(%v, %v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.originalURL, tt.args.userID)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Create(%v, %v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.originalURL, tt.args.userID)
		})
	}
}

func TestDBRepo_CreateAlreadyExists(t *testing.T) {
	type args struct {
		ctx         context.Context
		id          string
		originalURL string
		userID      string
		errorCode   string
	}
	tests := []struct {
		wantErr           assert.ErrorAssertionFunc
		args              args
		name              string
		want              string
		shouldBeCustomErr bool
	}{
		{
			name: "UniqueViolation, return err with existing id",
			args: args{
				ctx:         context.Background(),
				id:          "lelelele",
				originalURL: "http://ya.ru",
				userID:      "SomeUserID",
				errorCode:   pgerrcode.UniqueViolation,
			},
			want:              "lelelele",
			wantErr:           assert.Error,
			shouldBeCustomErr: true,
		},
		{
			name: "Some other error, return err without existing id",
			args: args{
				ctx:         context.Background(),
				id:          "lelelele",
				originalURL: "http://ya.ru",
				userID:      "SomeUserID",
				errorCode:   pgerrcode.DatabaseDropped,
			},
			want:              "",
			wantErr:           assert.Error,
			shouldBeCustomErr: false,
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
			mock.ExpectPrepare("INSERT INTO users").ExpectExec().
				WithArgs(tt.args.userID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectPrepare("INSERT INTO short_url").ExpectExec().
				WithArgs(tt.args.id, tt.args.originalURL, tt.args.userID).
				WillReturnError(&pgconn.PgError{Code: tt.args.errorCode})
			mock.ExpectPrepare("SELECT short_url FROM short_url").ExpectQuery().
				WithArgs(tt.args.originalURL).
				WillReturnRows(mock.NewRows([]string{"short_url"}).AddRow(tt.want))
			mock.ExpectRollback()
			got, err := D.Create(tt.args.ctx, tt.args.id, tt.args.originalURL, tt.args.userID)
			if !tt.wantErr(t, err, fmt.Sprintf("Create(%v, %v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.originalURL, tt.args.userID)) {
				return
			}
			if tt.shouldBeCustomErr {
				assert.ErrorIs(t, err, ErrAlreadyExists)
			}
			assert.Equalf(t, tt.want, got, "Create(%v, %v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.originalURL, tt.args.userID)
		})
	}
}

func TestDBRepo_Ping(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		args      args
		wantErr   assert.ErrorAssertionFunc
		name      string
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
		name        string
		args        args
		want        string
		wantDeleted bool
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				id:  "lelelele",
			},
			want:        "lelelele",
			wantDeleted: false,
		},
		{
			name: "not found",
			args: args{
				ctx: context.Background(),
				id:  "lelelele",
			},
			want:        "",
			wantDeleted: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			D := DBRepo{
				pool: db,
			}
			mock.ExpectPrepare("SELECT original_url, active FROM short_url").ExpectQuery().
				WithArgs(tt.args.id).
				WillReturnRows(mock.NewRows([]string{"original_url", "active"}).AddRow(tt.want, tt.wantDeleted))

			res, deleted := D.Read(tt.args.ctx, tt.args.id)
			assert.Equalf(t, tt.want, res, "Read(%v, %v)", tt.args.ctx, tt.args.id)
			assert.Equalf(t, !tt.wantDeleted, deleted, "Read(%v, %v)", tt.args.ctx, tt.args.id)
		})
	}
}

func TestNewDBRepo(t *testing.T) {
	type args struct {
		pool *sql.DB
	}
	tests := []struct {
		args args
		want *DBRepo
		name string
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
		ctx    context.Context
		URLs   map[string]models.ShortenBatchItemRequest
		userID string
	}
	tests := []struct {
		wantErr assert.ErrorAssertionFunc
		args    args
		name    string
		want    []models.ShortenBatchItemResponse
	}{
		{
			name: "Successful batch create",
			args: args{
				ctx: context.Background(),
				URLs: map[string]models.ShortenBatchItemRequest{
					"lele": {CorrelationID: "lelele", OriginalURL: "https://ya.ru"},
					"lolo": {CorrelationID: "lololo", OriginalURL: "https://yandex.ru"},
				},
				userID: "SomeUserID",
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
				userID: "SomeUserID",
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

			mock.ExpectPrepare("SELECT id FROM users").ExpectQuery().
				WithArgs(tt.args.userID).WillReturnRows(sqlmock.NewRows([]string{"id"}))

			mock.ExpectPrepare("INSERT INTO users").ExpectExec().
				WithArgs(tt.args.userID).
				WillReturnResult(sqlmock.NewResult(1, 1))

			mockStatement := mock.ExpectPrepare("INSERT INTO short_url")
			for range tt.args.URLs {
				mockStatement.
					ExpectExec().
					WillReturnResult(sqlmock.NewResult(1, 1))
			}
			mock.ExpectCommit()
			got, err := D.BatchCreate(tt.args.ctx, tt.args.URLs, tt.args.userID)
			if !tt.wantErr(t, err, fmt.Sprintf("BatchCreate(%v, %v, %v)", tt.args.ctx, tt.args.URLs, tt.args.userID)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "BatchCreate(%v, %v, %v)", tt.args.ctx, tt.args.URLs, tt.args.userID)
		})
	}
}

func TestDBRepo_GetShortURLByOriginalURL(t *testing.T) {
	type args struct {
		ctx         context.Context
		originalURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success",
			args: args{
				ctx:         context.Background(),
				originalURL: "https://ya.ru",
			},
			want: "lelelele",
		},
		{
			name: "not found",
			args: args{
				ctx:         context.Background(),
				originalURL: "https://google.com",
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
			mock.ExpectPrepare("SELECT short_url FROM short_url").ExpectQuery().
				WithArgs(tt.args.originalURL).
				WillReturnRows(mock.NewRows([]string{"short_url"}).AddRow(tt.want))
			res, err := D.GetShortURLByOriginalURL(tt.args.ctx, tt.args.originalURL)
			assert.Equalf(t, tt.want, res, "GetShortURLByOriginalURL(%v, %v)", tt.args.ctx, tt.args.originalURL)
			require.NoError(t, err)
		})
	}
}

func TestDBRepo_ReadByUserID(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
	}
	tests := []struct {
		wantErr assert.ErrorAssertionFunc
		args    args
		name    string
		want    []models.ShortURLsByUserResponse
	}{
		{
			name: "Successful batch read",
			args: args{
				ctx:    context.Background(),
				userID: "SomeUserID",
			},
			want: []models.ShortURLsByUserResponse{
				{
					ShortURL:    "lelele",
					OriginalURL: "http://ya.ru",
				},
				{
					ShortURL:    "lololo",
					OriginalURL: "http://yandex.ru",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Successful batch read of empty list",
			args: args{
				ctx:    context.Background(),
				userID: "SomeUserID",
			},
			want:    []models.ShortURLsByUserResponse{},
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
			var rs *sqlmock.Rows
			if len(tt.want) > 0 {
				rs = mock.NewRows([]string{"short_url", "original_url"}).
					AddRow(tt.want[0].ShortURL, tt.want[0].OriginalURL).
					AddRow(tt.want[1].ShortURL, tt.want[1].OriginalURL)
			} else {
				rs = mock.NewRows([]string{"short_url", "original_url"})
			}

			mock.ExpectPrepare("SELECT short_url, original_url FROM short_url").ExpectQuery().
				WithArgs(tt.args.userID).
				WillReturnRows(rs)
			res, err := D.ReadByUserID(tt.args.ctx, tt.args.userID)
			assert.Equalf(t, tt.want, res, "ReadByUserID(%v, %v)", tt.args.ctx, tt.args.userID)
			require.NoError(t, err)
		})
	}
}

func TestDBRepo_GetStats(t *testing.T) {
	tests := []struct {
		name string
		want models.ServiceStats
	}{
		{
			name: "success",
			want: models.ServiceStats{
				Users: 1337,
				URLs:  1338,
			},
		},
		{
			name: "failure",
			want: models.ServiceStats{
				Users: 1337,
				URLs:  1338,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			D := DBRepo{
				pool: db,
			}
			var rsUsers, rsUrls *sqlmock.Rows
			rsUsers = mock.NewRows([]string{"count"}).AddRow(tt.want.Users)
			rsUrls = mock.NewRows([]string{"count"}).AddRow(tt.want.URLs)

			mock.ExpectPrepare("SELECT count").ExpectQuery().WillReturnRows(rsUsers)
			mock.ExpectPrepare("SELECT count").ExpectQuery().WillReturnRows(rsUrls)
			res, err := D.GetStats(context.Background())

			assert.Equal(t, tt.want, res, "GetStats")
			require.NoError(t, err)
		})
	}
}
