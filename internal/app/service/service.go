// Package service stores both ShortURLServiceInterface and its main implementation.
package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/storage"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const shortURLIdLength = 8

// ErrShortURLNotFound is an error that will be returned in case the non-existing short URL is being requested
// by the user.
var ErrShortURLNotFound = errors.New("no urls found by the given id")

func generateID() string {
	bytesSlice := make([]byte, shortURLIdLength)
	for i := range bytesSlice {
		bytesSlice[i] = letters[rand.Intn(len(letters))]
	}
	return string(bytesSlice)
}

// ShortURLServiceInterface is an interface for the business-logic layer of the application.
type ShortURLServiceInterface interface {

	// Create creates the short URL by passed original URL and connects it with the user.
	Create(ctx context.Context, originalURL string, userID string) (string, error)

	// Read reads the original URL from the storage by passed ID, which is the ID of short URL.
	Read(ctx context.Context, id string) (string, bool, error)

	// Ping pings the required dependencies.
	Ping(ctx context.Context) error

	// BatchCreate creates the batch of short URLs using the batch of original URLs passed by user, connects all the
	// short URLs with this user.
	BatchCreate(ctx context.Context, requestData []models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error)

	// ReadByUserID Reads all the URLs created by the current user.
	ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error)

	// ScheduleDeletionOfBatch Schedules the batch of short URLs for the deletion.
	ScheduleDeletionOfBatch(shortURLs []models.ShortURLChannelMessage)

	// FlushDeletions marks some scheduled deletions as deleted in the storage.
	FlushDeletions()

	// GetStats returns the total number of users and shortened URLs stored in the service
	GetStats(ctx context.Context) (models.ServiceStats, error)
}

// ShortURLService is the structure that implements the ShortURLServiceInterface interface and performs as the main
// business-logic generalization for the short-url functionality.
type ShortURLService struct {
	repo             storage.Repository
	doneChan         chan struct{}
	deleteMsgChanIn  chan models.ShortURLChannelMessage
	deleteMsgChanOut chan string
}

// NewService initializes the new ShortURLService structure, using its dependencies as an input.
func NewService(repo storage.Repository, doneChan chan struct{}) ShortURLService {
	deleteMsgChanIn := make(chan models.ShortURLChannelMessage, config.Settings.DefaultChannelsBufferSize)
	deleteMsgChanOut := make(chan string, config.Settings.DefaultChannelsBufferSize)
	service := ShortURLService{repo: repo, deleteMsgChanIn: deleteMsgChanIn, deleteMsgChanOut: deleteMsgChanOut, doneChan: doneChan}
	go service.FlushDeletions()
	return service
}

// Create creates the short URL by passed original URL and connects it with the user. Generates the ID before saving to the storage.
func (s *ShortURLService) Create(ctx context.Context, originalURL string, userID string) (string, error) {
	var id string
	for {
		id = generateID()
		existingURLByID, _ := s.repo.Read(ctx, id)
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

// Read reads the original URL from the storage by passed ID, which is the ID of short URL.
func (s *ShortURLService) Read(ctx context.Context, id string) (string, bool, error) {
	originalURL, deleted := s.repo.Read(ctx, id)
	if originalURL == "" {
		return originalURL, false, ErrShortURLNotFound
	}
	return originalURL, deleted, nil
}

// FillRow saves the single row of file (cold-storage) to the storage (warm-storage).
func (s *ShortURLService) FillRow(ctx context.Context, originalURL string, shortURL string, userID string) error {
	_, err := s.repo.Create(ctx, shortURL, originalURL, userID)
	return err
}

// Ping pings the required dependencies.
func (s *ShortURLService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

// BatchCreate creates the batch of short URLs using the batch of original URLs passed by user, connects all the
// short URLs with this user.
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

// ReadByUserID Reads all the URLs created by the current user.
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

// FlushDeletions marks some scheduled deletions as deleted in the storage.
func (s *ShortURLService) FlushDeletions() {
	ticker := time.NewTicker(time.Duration(config.Settings.DeletionBufferFlushIntervalSeconds) * time.Second)

	var shortURLsToDelete []string

	for {
		select {
		case msg := <-s.deleteMsgChanOut:
			shortURLsToDelete = append(shortURLsToDelete, msg)
		case <-ticker.C:
			if len(shortURLsToDelete) == 0 {
				continue
			}
			err := s.repo.SetURLsInactive(context.TODO(), shortURLsToDelete)
			if err != nil {
				logger.Log.Warn("cannot delete URLs", zap.Error(err))
				continue
			}
			shortURLsToDelete = nil
		}
	}
}

// ScheduleDeletionOfBatch Schedules the batch of short URLs for the deletion. Uses FanOut + FanIn.
func (s *ShortURLService) ScheduleDeletionOfBatch(shortURLs []models.ShortURLChannelMessage) {
	s.deletionGenerator(shortURLs)
	channels := s.deletionFanOut()
	s.deletionFanIn(channels...)
}

func (s *ShortURLService) deletionGenerator(input []models.ShortURLChannelMessage) {
	go func() {
		for _, item := range input {
			select {
			case <-s.doneChan:
				return
			case s.deleteMsgChanIn <- item:
			}
		}
	}()
}

func (s *ShortURLService) deletionFanOut() []chan string {
	numWorkers := 10
	channels := make([]chan string, numWorkers)
	for i := 0; i < numWorkers; i++ {
		channels[i] = s.validateUser()
	}
	return channels
}

func (s *ShortURLService) validateUser() chan string {
	validateRes := make(chan string)
	go func() {
		defer close(validateRes)
		for data := range s.deleteMsgChanIn {
			currentUserID, err := s.repo.GetUserIDByShortURL(context.TODO(), data.ShortURL)
			if err != nil {
				logger.Log.Error("cannot get user ID", zap.String("shortURL", data.ShortURL), zap.Error(err))
			}
			if currentUserID == "" {
				logger.Log.Infof("Skipping URL %s - not found in storage", data.ShortURL)
				return
			}
			if currentUserID == data.UserID {
				select {
				case <-s.doneChan:
					return
				case validateRes <- data.ShortURL:
				}
			} else {
				logger.Log.Infof("Skipping URL %s - user is not the owner", data.ShortURL)
			}
		}
	}()
	return validateRes
}

func (s *ShortURLService) deletionFanIn(channels ...chan string) {
	for _, ch := range channels {
		chClosure := ch

		go func() {
			for data := range chClosure {
				select {
				case <-s.doneChan:
					return
				case s.deleteMsgChanOut <- data:
				}
			}
		}()

	}
}

// GetStats Returns the number of users and URLs registered in the service.
func (s *ShortURLService) GetStats(ctx context.Context) (models.ServiceStats, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return models.ServiceStats{}, err
	}
	return stats, nil
}
