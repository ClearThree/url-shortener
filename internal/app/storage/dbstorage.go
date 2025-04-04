package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBRepo struct {
	pool *sql.DB
}

func NewDBRepo(pool *sql.DB) *DBRepo {
	return &DBRepo{pool}
}

func (D DBRepo) Create(ctx context.Context, id string, originalURL string) (string, error) {
	createShortURLPreparedStmt, err := D.pool.PrepareContext(
		ctx, "INSERT INTO short_url (short_url, original_url) VALUES ($1, $2)")
	if err != nil {
		return "", err
	}
	_, err = createShortURLPreparedStmt.ExecContext(ctx, id, originalURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			logger.Log.Infof("OriginalURL %s already exists", originalURL)
			id, innerErr := D.GetShortURLByOriginalURL(ctx, originalURL)
			if innerErr != nil {
				return "", innerErr
			}
			err = NewErrAlreadyExists(ErrAlreadyExists, id)
			return id, err
		}
		return "", err
	}
	return id, nil
}

func (D DBRepo) Read(ctx context.Context, id string) string {
	readOriginalURLPreparedStmt, err := D.pool.PrepareContext(ctx, "SELECT original_url FROM short_url WHERE short_url = $1")
	if err != nil {
		return ""
	}
	result := readOriginalURLPreparedStmt.QueryRowContext(ctx, id)
	var originalURL string
	err = result.Scan(&originalURL)
	if err != nil {
		return ""
	}
	return originalURL

}

func (D DBRepo) GetShortURLByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	readOriginalURLPreparedStmt, err := D.pool.PrepareContext(ctx, "SELECT short_url FROM short_url WHERE original_url = $1")
	if err != nil {
		return "", err
	}
	result := readOriginalURLPreparedStmt.QueryRowContext(ctx, originalURL)
	var shortURL string
	err = result.Scan(&shortURL)
	if err != nil {
		return "", err
	}
	return shortURL, nil
}

func (D DBRepo) Ping(ctx context.Context) error {
	return D.pool.PingContext(ctx)
}

func (D DBRepo) BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest) ([]models.ShortenBatchItemResponse, error) {
	transaction, err := D.pool.Begin()
	if err != nil {
		return nil, err
	}
	createShortURLPreparedStmt, err := transaction.PrepareContext(
		ctx, "INSERT INTO short_url (short_url, original_url, correlation_id) VALUES ($1, $2, $3)")
	if err != nil {
		return nil, err
	}
	results := make([]models.ShortenBatchItemResponse, 0, len(URLs))
	for shortURL, data := range URLs {
		_, err = createShortURLPreparedStmt.ExecContext(ctx, shortURL, data.OriginalURL, data.CorrelationID)
		if err != nil {
			txErr := transaction.Rollback()
			if txErr != nil {
				logger.Log.Error(txErr.Error())
			}
			return nil, err
		}
		results = append(results, models.ShortenBatchItemResponse{CorrelationID: data.CorrelationID, ShortURL: shortURL})
	}
	txErr := transaction.Commit()
	if txErr != nil {
		logger.Log.Error(txErr.Error())
	}
	return results, nil
}
