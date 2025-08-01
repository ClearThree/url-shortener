// Package storage stores the interface and its in-memory implementation for the storage repository
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/clearthree/url-shortener/internal/app/models"
)

// ErrAlreadyExists is an error that returned  when one tries to shorten the URL that exists already in the storage.
var ErrAlreadyExists = errors.New("URL already exists")

// ErrAlreadyExistsExtended is a wrapper for ErrAlreadyExists to pass the existing short URL to the caller
// when the error happens. Implements
type ErrAlreadyExistsExtended struct {
	Err              error
	ExistingShortURL string
}

// NewErrAlreadyExists is the constructor that returns the new ErrAlreadyExistsExtended structure using the error as an input.
func NewErrAlreadyExists(err error, existingShortURL string) *ErrAlreadyExistsExtended {
	return &ErrAlreadyExistsExtended{err, existingShortURL}
}

// Unwrap unwraps the error - returns the original error itself.
func (e ErrAlreadyExistsExtended) Unwrap() error {
	return e.Err
}

// Error returns the string representation of an error message.
func (e ErrAlreadyExistsExtended) Error() string {
	return fmt.Sprintf("%s with %s short URL", e.Err.Error(), e.ExistingShortURL)
}

// Repository is the interface that all the storages must implement.
type Repository interface {

	// Create stores the single URL in the storage.
	Create(ctx context.Context, id string, originalURL string, userID string) (string, error)

	// Read reads the single original URL from the storage by its short ID.
	Read(ctx context.Context, id string) (string, bool)

	// Ping pings if the storage is alive.
	Ping(ctx context.Context) error

	// BatchCreate stores the batch of URLs in the storage.
	BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error)

	// ReadByUserID reads all the user-owned URLs from the storage.
	ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error)

	// GetUserIDByShortURL Reads the user ID of the short URL author from the storage.
	GetUserIDByShortURL(ctx context.Context, shortURL string) (string, error)

	// SetURLsInactive marks the URL as inactive in the storage.
	SetURLsInactive(ctx context.Context, shortURLs []string) error

	// GetStats returns the total number of users and shortened URLs stored in the storage
	GetStats(ctx context.Context) (*models.ServiceStats, error)
}

var memoryStorage map[string]string
var memoryIDsStorage map[string][]string
var memoryStorageUsersByURLs map[string]string
var memoryStorageDeactivatedURLs map[string]bool

// MemoryRepo struct implements the Repository interface as an in-memory storage. In-memory storage is a set of maps to
// store and obtain any needed data by O(1) complexity.
type MemoryRepo struct{}

// Create stores the single URL in the storage.
func (m MemoryRepo) Create(_ context.Context, id string, originalURL string, userID string) (string, error) {
	memoryStorage[id] = originalURL
	memoryStorageUsersByURLs[id] = userID
	currentShortURLs := memoryIDsStorage[userID]
	currentShortURLs = append(currentShortURLs, id)
	memoryIDsStorage[userID] = currentShortURLs
	return id, nil
}

// Read reads the single original URL from the storage by its short ID.
func (m MemoryRepo) Read(_ context.Context, id string) (string, bool) {
	originalURL, ok := memoryStorage[id]
	if !ok {
		return "", false
	}
	_, deleted := memoryStorageDeactivatedURLs[id]
	return originalURL, deleted
}

// Ping pings if the storage is alive. Just returns nil because it has no any infrastructural dependencies.
func (m MemoryRepo) Ping(_ context.Context) error {
	return nil
}

// BatchCreate stores the batch of URLs in the storage.
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

// ReadByUserID reads all the user-owned URLs from the storage.
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

// GetUserIDByShortURL Reads the user ID of the short URL author from the storage.
func (m MemoryRepo) GetUserIDByShortURL(_ context.Context, shortURL string) (string, error) {
	_, ok := memoryStorageDeactivatedURLs[shortURL]
	if ok {
		return "", nil
	}
	return memoryStorageUsersByURLs[shortURL], nil
}

// SetURLsInactive marks the URL as inactive in the storage.
func (m MemoryRepo) SetURLsInactive(_ context.Context, shortURLs []string) error {
	for _, shortURL := range shortURLs {
		memoryStorageDeactivatedURLs[shortURL] = true
	}
	return nil
}

// GetStats returns the total number of users and shortened URLs stored in the memory
func (m MemoryRepo) GetStats(_ context.Context) (*models.ServiceStats, error) {
	response := &models.ServiceStats{
		Users: len(memoryIDsStorage),
		URLs:  len(memoryStorage),
	}
	return response, nil
}

func init() {
	memoryStorage = make(map[string]string)
	memoryIDsStorage = make(map[string][]string)
	memoryStorageUsersByURLs = make(map[string]string)
	memoryStorageDeactivatedURLs = make(map[string]bool)
}
