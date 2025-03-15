package server

import (
	"github.com/clearthree/url-shortener/internal/app/logger"
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
	router.Use(logger.RequestLogger)
	router.Use(middleware.Recoverer)
	router.Post("/", createHandler.ServeHTTP)
	router.Post("/api/shorten", createJSONShortURLHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	return router
}

func Run(addr string) error {
	logger.Log.Infof("starting server at %s", addr)
	return http.ListenAndServe(addr, ShortenURLRouter())
}
