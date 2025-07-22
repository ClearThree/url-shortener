// Package models contains all the models used for json (de)serialization in handlers.
package models

import "context"

// ShortenRequest model is the model of input JSON used in CreateJSONShortURLHandler
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse model is the model of output JSON used in CreateJSONShortURLHandler
type ShortenResponse struct {
	Result string `json:"result"`
}

// ShortenBatchItemRequest is the model of input JSON used in BatchCreateShortURLHandler and ShortURLService
type ShortenBatchItemRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// ShortenBatchItemResponse is the model of output JSON used in BatchCreateShortURLHandler and ShortURLService
type ShortenBatchItemResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// ShortURLsByUserResponse is the model of output JSON used in GetAllURLsForUserHandler.
type ShortURLsByUserResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// ShortURLChannelMessage is the model of the message that the deletion handler sends to the channel.
type ShortURLChannelMessage struct {
	Ctx      context.Context
	ShortURL string
	UserID   string
}

// ServiceStats is the model of the message that the statistics handler responds with.
type ServiceStats struct {
	Users int `json:"users"` // the amount of users in the service
	URLs  int `json:"urls"`  // the amount of shortened URLs
}
