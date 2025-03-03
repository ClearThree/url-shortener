package service

import (
	"errors"
	"math/rand"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/storage"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const shortURLIdLength = 8

var ErrShortURLNotFound = errors.New("no urls found by the given id")

func generateID() string {
	bytesSlice := make([]byte, shortURLIdLength)
	for i := range bytesSlice {
		bytesSlice[i] = letters[rand.Intn(len(letters))]
	}
	return string(bytesSlice)
}

type Interface interface {
	Create(repo storage.Repository, originalURL string) (string, error)
	Read(repo storage.Repository, id string) (string, error)
}
type ShortURLService struct {
	repo storage.Repository
}

func NewService(repo storage.Repository) ShortURLService {
	return ShortURLService{repo: repo}
}

func (s *ShortURLService) Create(originalURL string) (string, error) {
	var id string
	for {
		id = generateID()
		existingURLByID := s.repo.Read(id)
		if existingURLByID == "" {
			break
		}
	}
	return config.Config.HostedOn.String() + s.repo.Create(id, originalURL), nil
}

func (s *ShortURLService) Read(id string) (string, error) {
	originalURL := s.repo.Read(id)
	if originalURL == "" {
		return originalURL, ErrShortURLNotFound
	}
	return originalURL, nil
}

var ShortURLServiceInstance = NewService(storage.MemoryRepo{})
