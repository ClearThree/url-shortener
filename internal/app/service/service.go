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
	Read(ctx context.Context, id string) (string, bool, error)
	Ping(ctx context.Context) error
	BatchCreate(ctx context.Context, requestData []models.ShortenBatchItemRequest, userID string) ([]models.ShortenBatchItemResponse, error)
	ReadByUserID(ctx context.Context, userID string) ([]models.ShortURLsByUserResponse, error)
	ScheduleDeletionOfBatch(shortURLs []models.ShortURLChannelMessage)
	FlushDeletions()
}
type ShortURLService struct {
	repo             storage.Repository
	doneChan         chan struct{}
	deleteMsgChanIn  chan models.ShortURLChannelMessage
	deleteMsgChanOut chan string
}

func NewService(repo storage.Repository, doneChan chan struct{}) ShortURLService {
	deleteMsgChanIn := make(chan models.ShortURLChannelMessage, config.Settings.DefaultChannelsBufferSize)
	deleteMsgChanOut := make(chan string, config.Settings.DefaultChannelsBufferSize)
	service := ShortURLService{repo: repo, deleteMsgChanIn: deleteMsgChanIn, deleteMsgChanOut: deleteMsgChanOut, doneChan: doneChan}
	go service.FlushDeletions()
	return service
}

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

func (s *ShortURLService) Read(ctx context.Context, id string) (string, bool, error) {
	originalURL, deleted := s.repo.Read(ctx, id)
	if originalURL == "" {
		return originalURL, false, ErrShortURLNotFound
	}
	return originalURL, deleted, nil
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
