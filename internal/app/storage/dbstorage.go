package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
)

type DBRepo struct {
	pool *sql.DB
}

func NewDBRepo(pool *sql.DB) *DBRepo {
	return &DBRepo{pool}
}

func (D DBRepo) Create(ctx context.Context, id string, originalURL string, userID string) (string, error) {
	transaction, err := D.pool.Begin()
	if err != nil {
		return "", err
	}
	createUserPreparedStmt, err := transaction.PrepareContext(
		ctx, "INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING")
	if err != nil {
		return "", err
	}
	_, userErr := createUserPreparedStmt.ExecContext(ctx, userID)
	if userErr != nil {
		txErr := transaction.Rollback()
		if txErr != nil {
			return "", txErr
		}
	}

	createShortURLPreparedStmt, err := transaction.PrepareContext(
		ctx, "INSERT INTO short_url (short_url, original_url, user_id) VALUES ($1, $2, $3)")
	if err != nil {
		return "", err
	}
	_, createErr := createShortURLPreparedStmt.ExecContext(ctx, id, originalURL, userID)
	if createErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(createErr, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			logger.Log.Infof("OriginalURL %s already exists", originalURL)
			existingID, innerErr := D.GetShortURLByOriginalURL(ctx, originalURL)
			if innerErr != nil {
				txErr := transaction.Rollback()
				if txErr != nil {
					return "", txErr
				}
				return "", innerErr
			}
			err = NewErrAlreadyExists(ErrAlreadyExists, existingID)
			transaction.Rollback()
			return existingID, err
		}
		txErr := transaction.Rollback()
		if txErr != nil {
			return "", txErr
		}
		return "", err
	}
	txErr := transaction.Commit()
	if txErr != nil {
		return "", txErr
	}
	return id, nil
}

func (D DBRepo) Read(ctx context.Context, id string) (string, bool) {
	readOriginalURLPreparedStmt, err := D.pool.PrepareContext(ctx, "SELECT original_url, active FROM short_url WHERE short_url = $1")
	if err != nil {
		return "", false
	}
	result := readOriginalURLPreparedStmt.QueryRowContext(ctx, id)
	var originalURL string
	var active bool
	err = result.Scan(&originalURL, &active)
	if err != nil {
		return "", false
	}
	return originalURL, !active

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

func (D DBRepo) BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error) {
	transaction, err := D.pool.Begin()
	if err != nil {
		return nil, err
	}
	createUserPreparedStmt, err := transaction.PrepareContext(
		ctx, "INSERT INTO users (id) VALUES ($1)")
	if err != nil {
		return nil, err
	}
	_, userErr := createUserPreparedStmt.ExecContext(ctx, userID)
	if userErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(userErr, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			logger.Log.Infof("UserID %s already exists", userID)
		}
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

func (D DBRepo) ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error) {
	readURLsByUserIDPreparedStmt, err := D.pool.PrepareContext(
		ctx, "SELECT short_url, original_url FROM short_url WHERE user_id = $1")
	if err != nil {
		return nil, err
	}
	rows, err := readURLsByUserIDPreparedStmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	results := make([]models.ShortURLsByUserResponse, 0)
	for rows.Next() {
		URL := new(models.ShortURLsByUserResponse)
		scanErr := rows.Scan(&URL.ShortURL, &URL.OriginalURL)
		if scanErr != nil {
			logger.Log.Error(scanErr.Error())
			return nil, scanErr
		}
		results = append(results, *URL)
	}
	return results, nil
}

func (D DBRepo) GetUserIDByShortURL(ctx context.Context, shortURL string) (string, error) {
	getUserIDByShortURLPreparedStmt, err := D.pool.PrepareContext(
		ctx, "SELECT user_id FROM short_url WHERE short_url = $1")
	if err != nil {
		return "", err
	}
	result := getUserIDByShortURLPreparedStmt.QueryRowContext(ctx, shortURL)
	var userID string
	err = result.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return userID, nil

}

func (D DBRepo) SetURLsInactive(ctx context.Context, shortURLs []string) error {
	var values []string
	var args []any
	for i, shortURL := range shortURLs {
		values = append(values, fmt.Sprintf("$%d", i+1))
		args = append(args, shortURL)
	}
	query := `
		  UPDATE short_url SET active = false, modified_at = NOW()
		  WHERE short_url in (` + strings.Join(values, ",") + `);`

	setURLsInactivePreparedStmt, err := D.pool.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	_, err = setURLsInactivePreparedStmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return err
}
