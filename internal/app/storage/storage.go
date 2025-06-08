package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/clearthree/url-shortener/internal/app/models"
)

var ErrAlreadyExists = errors.New("URL already exists")

type ErrAlreadyExistsExtended struct {
	Err              error
	ExistingShortURL string
}

func NewErrAlreadyExists(err error, existingShortURL string) *ErrAlreadyExistsExtended {
	return &ErrAlreadyExistsExtended{err, existingShortURL}
}

func (e ErrAlreadyExistsExtended) Unwrap() error {
	return e.Err
}

func (e ErrAlreadyExistsExtended) Error() string {
	return fmt.Sprintf("%s with %s short URL", e.Err.Error(), e.ExistingShortURL)
}

type Repository interface {
	Create(ctx context.Context, id string, originalURL string, userID string) (string, error)
	Read(ctx context.Context, id string) (string, bool)
	Ping(ctx context.Context) error
	BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error)
	ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error)
	GetUserIDByShortURL(ctx context.Context, shortURL string) (string, error)
	SetURLsInactive(ctx context.Context, shortURLs []string) error
}

var memoryStorage map[string]string
var memoryIDsStorage map[string][]string
var memoryStorageUsersByURLs map[string]string
var memoryStorageDeactivatedURLs map[string]bool

type MemoryRepo struct{}

func (m MemoryRepo) Create(_ context.Context, id string, originalURL string, userID string) (string, error) {
	memoryStorage[id] = originalURL
	memoryStorageUsersByURLs[id] = userID
	currentShortURLs := memoryIDsStorage[userID]
	currentShortURLs = append(currentShortURLs, id)
	memoryIDsStorage[userID] = currentShortURLs
	return id, nil
}

func (m MemoryRepo) Read(_ context.Context, id string) (string, bool) {
	originalURL, ok := memoryStorage[id]
	if !ok {
		return "", false
	}
	_, deleted := memoryStorageDeactivatedURLs[id]
	return originalURL, deleted
}

func (m MemoryRepo) Ping(_ context.Context) error {
	return nil
}

func (m MemoryRepo) BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error) {
	results := make([]models.ShortenBatchItemResponse, 0, len(URLs))
	for shortURL, data := range URLs {
		result, err := m.Create(ctx, shortURL, data.OriginalURL, userID)
		if err != nil {
			return nil, err
		}
		results = append(results, models.ShortenBatchItemResponse{CorrelationID: data.CorrelationID, ShortURL: result})
	}
	return results, nil
}
func (m MemoryRepo) ReadByUserID(_ context.Context, userID string) ([]models.ShortURLsByUserResponse, error) {
	currentShortURLs := memoryIDsStorage[userID]
	if len(currentShortURLs) == 0 {
		return nil, nil
	}
	result := make([]models.ShortURLsByUserResponse, 0)
	for _, shortURL := range currentShortURLs {
		_, deleted := memoryStorageDeactivatedURLs[shortURL]
		if deleted {
			continue
		}
		result = append(result, models.ShortURLsByUserResponse{
			ShortURL:    shortURL,
			OriginalURL: memoryStorage[shortURL],
		})
	}
	return result, nil
}

func (m MemoryRepo) GetUserIDByShortURL(_ context.Context, shortURL string) (string, error) {
	_, ok := memoryStorageDeactivatedURLs[shortURL]
	if ok {
		return "", nil
	}
	return memoryStorageUsersByURLs[shortURL], nil
}

func (m MemoryRepo) SetURLsInactive(_ context.Context, shortURLs []string) error {
	for _, shortURL := range shortURLs {
		memoryStorageDeactivatedURLs[shortURL] = true
	}
	return nil
}

func init() {
	memoryStorage = make(map[string]string)
	memoryIDsStorage = make(map[string][]string)
	memoryStorageUsersByURLs = make(map[string]string)
	memoryStorageDeactivatedURLs = make(map[string]bool)
}
