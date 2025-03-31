package storage

import (
	"context"
	"database/sql"
)

type DBRepo struct {
	pool *sql.DB
}

func NewDBRepo(pool *sql.DB) *DBRepo {
	return &DBRepo{pool}
}

func (D DBRepo) Create(ctx context.Context, id string, originalURL string) (string, error) {
	var createShortURLSQLQuery = "INSERT INTO short_url (id, original_url) VALUES ($1, $2)"
	_, err := D.pool.ExecContext(ctx, createShortURLSQLQuery, id, originalURL)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (D DBRepo) Read(ctx context.Context, id string) string {
	var readShortURLSQLQuery = "SELECT original_url FROM short_url WHERE id = $1"
	result := D.pool.QueryRowContext(ctx, readShortURLSQLQuery, id)
	var originalURL string
	err := result.Scan(&originalURL)
	if err != nil {
		return ""
	}
	return originalURL

}

func (D DBRepo) Ping(ctx context.Context) error {
	return D.pool.PingContext(ctx)
}
