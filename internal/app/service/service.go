package service

import (
	"errors"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/storage"
	"math/rand"
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

type ShortURLServiceInterface interface {
	Create(originalURL string) (string, error)
	Read(id string) (string, error)
	Ping() error
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
	result := config.Settings.HostedOn + s.repo.Create(id, originalURL)
	_, err := storage.FSWrapper.Create(id, originalURL)
	return result, err
}

func (s *ShortURLService) Read(id string) (string, error) {
	originalURL := s.repo.Read(id)
	if originalURL == "" {
		return originalURL, ErrShortURLNotFound
	}
	return originalURL, nil
}

func (s *ShortURLService) FillRow(originalURL string, shortURL string) error {
	s.repo.Create(shortURL, originalURL)
	return nil
}

func (s *ShortURLService) Ping() error {
	return s.repo.Ping()
}
