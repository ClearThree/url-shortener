package handlers

import (
	"encoding/json"
	"errors"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/models"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clearthree/url-shortener/internal/app/service"
)

const maxPayloadSize = 1024 * 1024

func isURL(payload string) bool {
	parsedURL, err := url.Parse(payload)
	if err != nil {
		return false
	}
	return parsedURL.Scheme == "https" || parsedURL.Scheme == "http"
}

type CreateShortURLHandler struct{}

func (create CreateShortURLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	//if contentType := request.Header.Get("Content-Type"); !strings.Contains(contentType, "text/plain") {
	//	http.Error(writer, "Only text/plain content type is allowed", http.StatusBadRequest)
	//	return
	//}  // Внезапно, ломает тесты 8 инкремента
	contentLength, err := strconv.Atoi(request.Header.Get("Content-Length"))
	if err != nil {
		http.Error(writer, "Content-Length header is invalid, should be integer", http.StatusBadRequest)
		return
	}
	if contentLength > maxPayloadSize {
		http.Error(writer, "Content is too large", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()
	payload, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Couldn't read the request body", http.StatusBadRequest)
		return
	}
	if len(payload) == 0 {
		http.Error(writer, "Please provide an url", http.StatusBadRequest)
		return
	}
	payloadString := string(payload)
	if !isURL(payloadString) {
		http.Error(writer, "The provided payload is not a valid URL", http.StatusBadRequest)
		return
	}
	id, err := service.ShortURLServiceInstance.Create(payloadString)
	if err != nil {
		http.Error(writer, "Couldn't create short url", http.StatusBadRequest)
		return
	}
	writer.Header().Add("Content-Type", "text/plain")
	writer.WriteHeader(http.StatusCreated)
	_, err = writer.Write([]byte(id))
	if err != nil {
		http.Error(writer, "Couldn't write the response body", http.StatusBadRequest)
		return
	}
}

type RedirectToOriginalURLHandler struct{}

func (redirect RedirectToOriginalURLHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		http.Error(writer, "Please provide the short url ID", http.StatusBadRequest)
		return
	}
	originalURL, err := service.ShortURLServiceInstance.Read(id)
	if err != nil {
		if errors.Is(err, service.ErrShortURLNotFound) {
			http.Error(writer, "Short url not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "Something went wrong", http.StatusBadRequest)
		return
	}

	http.Redirect(writer, request, originalURL, http.StatusTemporaryRedirect)
}

type CreateJSONShortURLHandler struct{}

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
	id, err := service.ShortURLServiceInstance.Create(requestData.URL)
	if err != nil {
		http.Error(writer, "Couldn't create short url", http.StatusBadRequest)
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(writer)
	responseData := models.ShortenResponse{Result: id}
	if err := enc.Encode(responseData); err != nil {
		logger.Log.Debugf("Error encoding response: %s", err)
		return
	}
}
