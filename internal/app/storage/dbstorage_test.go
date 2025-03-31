package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
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
			mock.ExpectExec("INSERT INTO short_url").
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
			mock.ExpectQuery("SELECT original_url FROM short_url").
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
