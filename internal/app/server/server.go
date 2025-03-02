package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"

	"github.com/clearthree/url-shortener/internal/app/handlers"
)

func ShortenURLRouter() chi.Router {
	var createHandler handlers.CreateShortURLHandler
	var redirectHandler handlers.RedirectToOriginalURLHandler

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	//router.Use(middleware.AllowContentType("text/plain"))
	router.Post("/", createHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	return router
}

func Run() error {
	return http.ListenAndServe(`localhost:8080`, ShortenURLRouter())
}
