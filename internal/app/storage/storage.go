package storage

import (
	"context"
	"github.com/clearthree/url-shortener/internal/app/models"
)

type Repository interface {
	Create(ctx context.Context, string, originalURL string) (string, error)
	Read(ctx context.Context, id string) string
	Ping(ctx context.Context) error
	BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest) ([]models.ShortenBatchItemResponse, error)
}

var memoryStorage map[string]string

type MemoryRepo struct{}

func (m MemoryRepo) Create(_ context.Context, id string, originalURL string) (string, error) {
	memoryStorage[id] = originalURL
	return id, nil
}

func (m MemoryRepo) Read(_ context.Context, id string) string {
	originalURL, ok := memoryStorage[id]
	if !ok {
		return ""
	}
	return originalURL
}

func (m MemoryRepo) Ping(_ context.Context) error {
	return nil
}

func (m MemoryRepo) BatchCreate(ctx context.Context, URLs map[string]models.ShortenBatchItemRequest) ([]models.ShortenBatchItemResponse, error) {
	results := make([]models.ShortenBatchItemResponse, 0, len(URLs))
	for shortURL, data := range URLs {
		result, err := m.Create(ctx, shortURL, data.OriginalURL)
		if err != nil {
			return nil, err
		}
		results = append(results, models.ShortenBatchItemResponse{CorrelationID: data.CorrelationID, ShortURL: result})
	}
	return results, nil
}

func init() {
	memoryStorage = make(map[string]string)
}
