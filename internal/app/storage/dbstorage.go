package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/models"
)

// DBRepo is the Database-based implementation of Repository interface.
type DBRepo struct {
	pool *sql.DB
}

// NewDBRepo is a constructor for the new DBRepo structure instance.
func NewDBRepo(pool *sql.DB) *DBRepo {
	return &DBRepo{pool}
}

// Create stores the single URL in the database.
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
			txErr := transaction.Rollback()
			if txErr != nil {
				return "", txErr
			}
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

// Read reads the single original URL from the database by its short ID.
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

// GetShortURLByOriginalURL takes the short URL from the database by the provided original URL.
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

// Ping pings if the database is alive.
func (D DBRepo) Ping(ctx context.Context) error {
	return D.pool.PingContext(ctx)
}

// BatchCreate stores the batch of URLs in the database.
func (D DBRepo) BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error) {
	transaction, err := D.pool.Begin()
	if err != nil {
		return nil, err
	}
	existingUserPreparedStmt, err := transaction.PrepareContext(ctx,
		"SELECT id FROM users WHERE id = $1")
	if err != nil {
		return nil, err
	}
	var existingUserID string
	userRow := existingUserPreparedStmt.QueryRowContext(ctx, userID)
	if userRow.Err() != nil {
		return nil, userRow.Err()
	}

	if err = userRow.Scan(&existingUserID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if existingUserID == "" {
		createUserPreparedStmt, prepareErr := transaction.PrepareContext(
			ctx, "INSERT INTO users (id) VALUES ($1)")
		if prepareErr != nil {
			return nil, prepareErr
		}
		_, userErr := createUserPreparedStmt.ExecContext(ctx, userID)
		if userErr != nil {
			return nil, userErr
		}
	}

	createShortURLPreparedStmt, err := transaction.PrepareContext(
		ctx, "INSERT INTO short_url (short_url, original_url, correlation_id, user_id) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return nil, err
	}
	results := make([]models.ShortenBatchItemResponse, len(URLs))
	cnt := 0
	for shortURL, data := range URLs {
		_, err = createShortURLPreparedStmt.ExecContext(ctx, shortURL, data.OriginalURL, data.CorrelationID, userID)
		if err != nil {
			txErr := transaction.Rollback()
			if txErr != nil {
				logger.Log.Error(txErr.Error())
			}
			return nil, err
		}
		results[cnt] = models.ShortenBatchItemResponse{CorrelationID: data.CorrelationID, ShortURL: shortURL}
		cnt++
	}
	txErr := transaction.Commit()
	if txErr != nil {
		logger.Log.Error(txErr.Error())
	}
	return results, nil
}

// ReadByUserID reads all the user-owned URLs from the database.
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
		URL := models.ShortURLsByUserResponse{}
		scanErr := rows.Scan(&URL.ShortURL, &URL.OriginalURL)
		if scanErr != nil {
			logger.Log.Error(scanErr.Error())
			return nil, scanErr
		}
		results = append(results, URL)
	}
	return results, nil
}

// GetUserIDByShortURL Reads the user ID of the short URL author from the database.
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

// SetURLsInactive marks the URL as inactive in the database.
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

// GetStats returns the total number of users and shortened URLs stored in the database
func (D DBRepo) GetStats(ctx context.Context) (models.ServiceStats, error) {
	usersCountPreparedStmt, err := D.pool.PrepareContext(
		ctx, "SELECT count(*) FROM users")
	if err != nil {
		return models.ServiceStats{}, err
	}
	result := usersCountPreparedStmt.QueryRowContext(ctx)
	var usersCount int
	err = result.Scan(&usersCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ServiceStats{Users: 0, URLs: 0}, nil
		}
		return models.ServiceStats{}, err
	}

	URLsCountPreparedStatement, err := D.pool.PrepareContext(
		ctx, "SELECT count(*) FROM short_url")
	if err != nil {
		return models.ServiceStats{}, err
	}
	result = URLsCountPreparedStatement.QueryRowContext(ctx)
	var URLsCount int
	err = result.Scan(&URLsCount)
	if err != nil {
		return models.ServiceStats{}, err
	}

	response := models.ServiceStats{
		Users: usersCount,
		URLs:  URLsCount,
	}
	return response, nil
}
