package server

import (
	"errors"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/middlewares"
	"github.com/clearthree/url-shortener/internal/app/service"
	"github.com/clearthree/url-shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"

	"github.com/clearthree/url-shortener/internal/app/handlers"
)

func ShortenURLRouter() chi.Router {
	var createHandler handlers.CreateShortURLHandler
	var createJSONShortURLHandler handlers.CreateJSONShortURLHandler
	var redirectHandler handlers.RedirectToOriginalURLHandler

	router := chi.NewRouter()
	router.Use(middlewares.RequestLogger)
	router.Use(middlewares.GzipMiddleware)
	router.Use(middleware.Recoverer)
	router.Post("/", createHandler.ServeHTTP)
	router.Post("/api/shorten", createJSONShortURLHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	return router
}

func Run(addr string) error {
	logger.Log.Infof("starting server at %s", addr)
	err := prefillMemory()
	if err != nil {
		return err
	}
	logger.Log.Info("Memory prefilled from file")
	err = storage.FSWrapper.Open()
	if err != nil {
		return err
	}
	defer storage.FSWrapper.Close()
	return http.ListenAndServe(addr, ShortenURLRouter())
}

func prefillMemory() error {

	for {
		row, err := storage.FSWrapper.ReadNextLine()
		if err != nil {
			if errors.Is(err, storage.ErrorFileReadCompletely) {
				break
			} else {
				return err
			}
		}
		fillingError := service.ShortURLServiceInstance.FillRow(row.OriginalURL, row.ShortURL)
		if fillingError != nil {
			return fillingError
		}
	}
	return nil
}
