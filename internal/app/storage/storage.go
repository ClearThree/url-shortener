package storage

type Repository interface {
	Create(id string, originalURL string) string
	Read(id string) string
	Ping() error
}

var memoryStorage map[string]string

type MemoryRepo struct{}

func (m MemoryRepo) Create(id string, originalURL string) string {
	memoryStorage[id] = originalURL
	return id
}

func (m MemoryRepo) Read(id string) string {
	originalURL, ok := memoryStorage[id]
	if !ok {
		return ""
	}
	return originalURL
}

func (m MemoryRepo) Ping() error {
	return nil
}

func init() {
	memoryStorage = make(map[string]string)
}
