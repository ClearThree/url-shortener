package server

import (
	"net/http"

	"github.com/clearthree/url-shortener/internal/app/handlers"
)

func Run() error {
	var createHandler handlers.CreateShortURLHandler
	var redirectHandler handlers.RedirectToOriginalURLHandler

	mux := http.NewServeMux()
	mux.Handle("/", createHandler)
	mux.Handle("/{id}", redirectHandler)
	return http.ListenAndServe(`:8080`, mux)
}
