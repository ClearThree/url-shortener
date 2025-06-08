package handlers

import (
	"github.com/clearthree/url-shortener/internal/app/service"
	"github.com/clearthree/url-shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Example() {

	// Initialize dependencies
	memoryRepo := storage.MemoryRepo{}
	doneChan := make(chan struct{})
	shortURLService := service.NewService(memoryRepo, doneChan)

	// Initialize the handler structures
	createHandler := NewCreateShortURLHandler(&shortURLService)
	createJSONShortURLHandler := NewCreateJSONShortURLHandler(&shortURLService)
	redirectHandler := NewRedirectToOriginalURLHandler(&shortURLService)
	pingHandler := NewPingHandler(&shortURLService)
	batchCreateHandler := NewBatchCreateShortURLHandler(&shortURLService)
	getAllUrlsByUserHandler := NewGetAllURLsForUserHandler(&shortURLService)
	deleteBatchOfURLsHandler := NewDeleteBatchOfURLsHandler(&shortURLService)

	// Initialize mux and attach handlers to serve routes
	router := chi.NewRouter()
	router.Post("/", createHandler.ServeHTTP)
	router.Post("/api/shorten", createJSONShortURLHandler.ServeHTTP)
	router.Post("/api/shorten/batch", batchCreateHandler.ServeHTTP)
	router.Get("/api/user/urls", getAllUrlsByUserHandler.ServeHTTP)
	router.Delete("/api/user/urls", deleteBatchOfURLsHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	router.Get("/ping", pingHandler.ServeHTTP)

	// Start the server
	err := http.ListenAndServe("localhost:8080", router)
	if err != nil {
		panic(err)
	}
}
