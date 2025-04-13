package service

import (
	"context"
	"errors"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/models"
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
	Create(ctx context.Context, originalURL string, userID string) (string, error)
	Read(ctx context.Context, id string) (string, error)
	Ping(ctx context.Context) error
	BatchCreate(ctx context.Context, requestData []models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error)
	ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error)
}
type ShortURLService struct {
	repo storage.Repository
}

func NewService(repo storage.Repository) ShortURLService {
	return ShortURLService{repo: repo}
}

func (s *ShortURLService) Create(ctx context.Context, originalURL string, userID string) (string, error) {
	var id string
	for {
		id = generateID()
		existingURLByID := s.repo.Read(ctx, id)
		if existingURLByID == "" {
			break
		}
	}
	shortURL, err := s.repo.Create(ctx, id, originalURL, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrAlreadyExists) {
			return "", err
		}
	}
	result := config.Settings.HostedOn + shortURL
	_, fsWrapperErr := storage.FSWrapper.Create(id, originalURL, userID)
	if fsWrapperErr != nil {
		return "", fsWrapperErr
	}
	return result, err
}

func (s *ShortURLService) Read(ctx context.Context, id string) (string, error) {
	originalURL := s.repo.Read(ctx, id)
	if originalURL == "" {
		return originalURL, ErrShortURLNotFound
	}
	return originalURL, nil
}

func (s *ShortURLService) FillRow(ctx context.Context, originalURL string, shortURL string, userID string) error {
	_, err := s.repo.Create(ctx, shortURL, originalURL, userID)
	return err
}

func (s *ShortURLService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *ShortURLService) BatchCreate(
	ctx context.Context, requestData []models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error) {
	URLs := make(map[string]models.ShortenBatchItemRequest)
	for _, item := range requestData {
		shortURL := generateID()
		URLs[shortURL] = item
	}
	result, err := s.repo.BatchCreate(ctx, URLs, userID)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(result); i++ {
		data := &result[i]
		data.ShortURL = config.Settings.HostedOn + data.ShortURL
	}
	_, err = storage.FSWrapper.BatchCreate(URLs, userID)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ShortURLService) ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error) {
	result, err := s.repo.ReadByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(result); i++ {
		data := &result[i]
		data.ShortURL = config.Settings.HostedOn + data.ShortURL
	}
	return result, err
}
