// Package handlers stores all the structures that implement handlers.
// Each handler must be initialized by creating a structure with its constructor-method.
// All constructor methods accept the required dependencies, which are used later in the handler function.
// Handler functions ServeHTTP must be implemented to comply the IHandler interface
// and should be registered in web application.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/middlewares"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/storage"

	"github.com/clearthree/url-shortener/internal/app/service"
)

// maxPayloadSize - is the maximum size of payload that the server can process in the request.
const maxPayloadSize = 1024 * 1024

func isURL(payload string) bool {
	parsedURL, err := url.Parse(payload)
	if err != nil {
		return false
	}
	return parsedURL.Scheme == "https" || parsedURL.Scheme == "http"
}

// IHandler is the interface for all handler-structures
type IHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// CreateShortURLHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to create the short URL from the passed URL.
type CreateShortURLHandler struct {
	service service.ShortURLServiceInterface
}

// NewCreateShortURLHandler is a constructor function that returns a pointer
// to the freshly created CreateShortURLHandler structure.
func NewCreateShortURLHandler(service service.ShortURLServiceInterface) *CreateShortURLHandler {
	return &CreateShortURLHandler{service: service}
}

// ServeHTTP Serves as handler function. Creates a short URL for the passed original URL.
// Accepts text/plain request body that contains a valid URL.
// Maximal body size is defined with maxPayloadSize constant.
// Responds with text/plain body that contains a valid short URL.
func (create CreateShortURLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); !(strings.Contains(contentType, "text/plain") ||
		strings.Contains(contentType, "application/x-gzip")) {
		logger.Log.Warnf("Invalid content type: %s", contentType)
		http.Error(writer, "Only text/plain or application/x-gzip content types are allowed", http.StatusBadRequest)
		return
	}
	contentLength, err := strconv.Atoi(request.Header.Get("Content-Length"))
	if err != nil {
		logger.Log.Warnf("Invalid content length: %s", request.Header.Get("Content-Length"))
		http.Error(writer, "Content-Length header is invalid, should be integer", http.StatusBadRequest)
		return
	}
	if contentLength > maxPayloadSize {
		logger.Log.Warnf("Content is too large: %d", contentLength)
		http.Error(writer, "Content is too large", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()
	payload, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Log.Warn("Couldn't read the request body")
		http.Error(writer, "Couldn't read the request body", http.StatusBadRequest)
		return
	}
	if len(payload) == 0 {
		logger.Log.Warn("Couldn't read the request body")
		http.Error(writer, "Please provide an url", http.StatusBadRequest)
		return
	}
	payloadString := string(payload)
	if !isURL(payloadString) {
		logger.Log.Warnf("Invalid url: %s", payloadString)
		http.Error(writer, "The provided payload is not a valid URL", http.StatusBadRequest)
		return
	}
	userID := request.Header.Get(middlewares.UserIDHeaderName)
	id, err := create.service.Create(request.Context(), payloadString, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			create.writeResponse(writer, http.StatusConflict, id)
			return
		}
		logger.Log.Warnf("Failed to create short URL %v", err)
		http.Error(writer, "Couldn't create short url", http.StatusBadRequest)
		return
	}
	create.writeResponse(writer, http.StatusCreated, id)
}

func (create CreateShortURLHandler) writeResponse(writer http.ResponseWriter, statusCode int, body string) {
	writer.Header().Add("Content-Type", "text/plain")
	writer.WriteHeader(statusCode)
	_, err := writer.Write([]byte(body))
	if err != nil {
		http.Error(writer, "Couldn't write the response body", http.StatusBadRequest)
	}
}

// RedirectToOriginalURLHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to redirect the user to the original URL extracted using the passed short URL.
type RedirectToOriginalURLHandler struct {
	service service.ShortURLServiceInterface
}

// NewRedirectToOriginalURLHandler is a constructor function that returns a pointer
// to the freshly created RedirectToOriginalURLHandler structure.
func NewRedirectToOriginalURLHandler(service service.ShortURLServiceInterface) *RedirectToOriginalURLHandler {
	return &RedirectToOriginalURLHandler{service: service}
}

// ServeHTTP Serves as handler function. Extracts the original URL from the storage using passed short URL,
// then responds with temporary redirection to the extracted URL.
func (redirect RedirectToOriginalURLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		http.Error(writer, "Please provide the short url ID", http.StatusBadRequest)
		return
	}
	originalURL, deleted, err := redirect.service.Read(request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrShortURLNotFound) {
			http.Error(writer, "Short url not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "Something went wrong", http.StatusBadRequest)
		return
	}
	if deleted {
		writer.WriteHeader(http.StatusGone)
		return
	}

	http.Redirect(writer, request, originalURL, http.StatusTemporaryRedirect)
}

// CreateJSONShortURLHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to create short URL for the original URL accepted as JSON.
type CreateJSONShortURLHandler struct {
	service service.ShortURLServiceInterface
}

// NewCreateJSONShortURLHandler is a constructor function that returns a pointer
// to the freshly created CreateJSONShortURLHandler structure.
func NewCreateJSONShortURLHandler(service service.ShortURLServiceInterface) *CreateJSONShortURLHandler {
	return &CreateJSONShortURLHandler{service: service}
}

// ServeHTTP Serves as handler function.
// Creates the short URL for the passed original URL as a JSON, specified in models.ShortenRequest.
// Responds with a JSON document, specified in models.ShortenResponse.
func (create CreateJSONShortURLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		http.Error(writer, "Only application/json content type is allowed", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var requestData models.ShortenRequest
	dec := json.NewDecoder(request.Body)
	if err := dec.Decode(&requestData); err != nil {
		logger.Log.Debugf("Couldn't decode the request body: %s", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(requestData.URL) == 0 {
		http.Error(writer, "Please provide an url", http.StatusBadRequest)
		return
	}
	if !isURL(requestData.URL) {
		http.Error(writer, "The provided payload is not a valid URL", http.StatusBadRequest)
		return
	}
	userID := request.Header.Get(middlewares.UserIDHeaderName)
	id, err := create.service.Create(request.Context(), requestData.URL, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			create.writeResponse(writer, http.StatusConflict, id)
			return
		}
		http.Error(writer, "Couldn't create short url", http.StatusBadRequest)
		return
	}
	create.writeResponse(writer, http.StatusCreated, id)
}

func (create CreateJSONShortURLHandler) writeResponse(writer http.ResponseWriter, statusCode int, body string) {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	enc := json.NewEncoder(writer)
	responseData := models.ShortenResponse{Result: body}
	if err := enc.Encode(responseData); err != nil {
		logger.Log.Debugf("Error encoding response: %s", err)
		http.Error(writer, "Error encoding response body", http.StatusInternalServerError)
		return
	}
}

// PingHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to ping the dependencies of a service.
type PingHandler struct {
	service service.ShortURLServiceInterface
}

// NewPingHandler is a constructor function that returns a pointer
// to the freshly created PingHandler structure.
func NewPingHandler(service service.ShortURLServiceInterface) *PingHandler {
	return &PingHandler{service: service}
}

// ServeHTTP Serves as handler function.
// Responds with an error if any of the dependencies is not working as expected.
func (ping PingHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	err := ping.service.Ping(request.Context())
	if err != nil {
		http.Error(writer, "Database is not available", http.StatusInternalServerError)
	}
}

// BatchCreateShortURLHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to create the batch of short URLs for the given batch of original URLs.
type BatchCreateShortURLHandler struct {
	service service.ShortURLServiceInterface
}

// NewBatchCreateShortURLHandler is a constructor function that returns a pointer
// to the freshly created BatchCreateShortURLHandler structure.
func NewBatchCreateShortURLHandler(service service.ShortURLServiceInterface) *BatchCreateShortURLHandler {
	return &BatchCreateShortURLHandler{service: service}
}

// ServeHTTP Serves as handler function.
// Accepts JSON which is a list of models.ShortenBatchItemRequest objects, creates the short URL for each and
// responds with a JSON which is a list of models.ShortenBatchItemResponse objects.
func (create BatchCreateShortURLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		http.Error(writer, "Only application/json content type is allowed", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var requestData []models.ShortenBatchItemRequest
	dec := json.NewDecoder(request.Body)
	if err := dec.Decode(&requestData); err != nil {
		logger.Log.Debugf("Couldn't decode the request body: %s", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(requestData) == 0 {
		http.Error(writer, "Please provide a batch of URLs", http.StatusBadRequest)
		return
	}
	for _, requestItem := range requestData {
		if !isURL(requestItem.OriginalURL) {
			http.Error(writer, "One of the provided items is not a valid URL", http.StatusBadRequest)
			return
		}
	}
	userID := request.Header.Get(middlewares.UserIDHeaderName)
	results, err := create.service.BatchCreate(request.Context(), requestData, userID)
	if err != nil {
		http.Error(writer, "Couldn't create short url", http.StatusBadRequest)
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(writer)
	if err = enc.Encode(results); err != nil {
		logger.Log.Debugf("Error encoding response: %s", err)
		return
	}
}

// GetAllURLsForUserHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to return all the URLs created by authorized user.
type GetAllURLsForUserHandler struct {
	service service.ShortURLServiceInterface
}

// NewGetAllURLsForUserHandler is a constructor function that returns a pointer
// to the freshly created GetAllURLsForUserHandler structure.
func NewGetAllURLsForUserHandler(service service.ShortURLServiceInterface) *GetAllURLsForUserHandler {
	return &GetAllURLsForUserHandler{service: service}
}

// ServeHTTP Serves as handler function.
// Responds with a JSON which is a list of models.ShortURLsByUserResponse objects.
func (getHandler GetAllURLsForUserHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	userID := request.Header.Get(middlewares.UserIDHeaderName)
	results, err := getHandler.service.ReadByUserID(request.Context(), userID)
	if err != nil {
		http.Error(writer, "Couldn't read all the urls for user", http.StatusBadRequest)
		return
	}
	if len(results) == 0 {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(writer)
	if err = enc.Encode(results); err != nil {
		logger.Log.Debugf("Error encoding response: %s", err)
		return
	}
}

// DeleteBatchOfURLsHandler is a structure to store dependencies and
// implement ServeHTTP Handler function to return all the URLs created by authorized user.
type DeleteBatchOfURLsHandler struct {
	service service.ShortURLServiceInterface
}

// NewDeleteBatchOfURLsHandler is a constructor function that returns a pointer
// to the freshly created DeleteBatchOfURLsHandler structure.
func NewDeleteBatchOfURLsHandler(service service.ShortURLServiceInterface) *DeleteBatchOfURLsHandler {
	return &DeleteBatchOfURLsHandler{service: service}
}

// ServeHTTP Serves as handler function.
// Accepts the JSON-formatted list of short URL IDs, that should be deleted.
// Schedules the deletion of passed URLs. The URL will be deleted in some time after the response (not instantly).
// The URL will be deleted if it was created by the same user that tries to delete it.
func (delete DeleteBatchOfURLsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		http.Error(writer, "Only application/json content type is allowed", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var requestData []string
	dec := json.NewDecoder(request.Body)
	if err := dec.Decode(&requestData); err != nil {
		logger.Log.Debugf("Couldn't decode the request body: %s", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(requestData) == 0 {
		http.Error(writer, "Please provide a batch of shortURLs", http.StatusBadRequest)
		return
	}

	userID := request.Header.Get(middlewares.UserIDHeaderName)
	requestPrepared := make([]models.ShortURLChannelMessage, len(requestData))
	for i, requestItem := range requestData {
		requestPrepared[i] = models.ShortURLChannelMessage{
			Ctx:      request.Context(),
			ShortURL: requestItem,
			UserID:   userID,
		}
	}
	go delete.service.ScheduleDeletionOfBatch(requestPrepared)
	writer.WriteHeader(http.StatusAccepted)
}
