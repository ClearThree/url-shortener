package storage

import "context"

type Repository interface {
	Create(ctx context.Context, string, originalURL string) (string, error)
	Read(ctx context.Context, id string) string
	Ping(ctx context.Context) error
}

var memoryStorage map[string]string

type MemoryRepo struct{}

func (m MemoryRepo) Create(ctx context.Context, id string, originalURL string) (string, error) {
	memoryStorage[id] = originalURL
	return id, nil
}

func (m MemoryRepo) Read(ctx context.Context, id string) string {
	originalURL, ok := memoryStorage[id]
	if !ok {
		return ""
	}
	return originalURL
}

func (m MemoryRepo) Ping(ctx context.Context) error {
	return nil
}

func init() {
	memoryStorage = make(map[string]string)
}
