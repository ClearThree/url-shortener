// Package server is the package with all the tools required to run the server
package server

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/handlers"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/middlewares"
	"github.com/clearthree/url-shortener/internal/app/service"
	"github.com/clearthree/url-shortener/internal/app/storage"
)

// Pool is a global connection pool variable.
var Pool *sql.DB
var shortURLService service.ShortURLService

// ShortenURLRouter is the function to create the router along with all the business-logic implementations.
func ShortenURLRouter(pool *sql.DB, doneChan chan struct{}) chi.Router {
	if pool == nil {
		shortURLService = service.NewService(storage.MemoryRepo{}, doneChan)
	} else {
		shortURLService = service.NewService(storage.NewDBRepo(pool), doneChan)
	}
	var shortURLServiceDB = service.NewService(storage.NewDBRepo(pool), doneChan)

	var createHandler = handlers.NewCreateShortURLHandler(&shortURLService)
	var createJSONShortURLHandler = handlers.NewCreateJSONShortURLHandler(&shortURLService)
	var redirectHandler = handlers.NewRedirectToOriginalURLHandler(&shortURLService)
	var pingHandler = handlers.NewPingHandler(&shortURLServiceDB)
	var batchCreateHandler = handlers.NewBatchCreateShortURLHandler(&shortURLService)
	var getAllUrlsByUserHandler = handlers.NewGetAllURLsForUserHandler(&shortURLService)
	var deleteBatchOfURLsHandler = handlers.NewDeleteBatchOfURLsHandler(&shortURLService)
	var getStatsHandler = handlers.NewGetStatsHandler(&shortURLService)

	router := chi.NewRouter()
	router.Use(middlewares.RequestLogger)
	router.Use(middlewares.AuthMiddleware)
	router.Use(middlewares.GzipMiddleware)
	router.Use(middleware.Recoverer)
	router.Post("/", createHandler.ServeHTTP)
	router.Post("/api/shorten", createJSONShortURLHandler.ServeHTTP)
	router.Post("/api/shorten/batch", batchCreateHandler.ServeHTTP)
	router.Get("/api/user/urls", getAllUrlsByUserHandler.ServeHTTP)
	router.Delete("/api/user/urls", deleteBatchOfURLsHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	router.Get("/ping", pingHandler.ServeHTTP)

	router.Route("/api/internal", func(r chi.Router) {
		internalRoutesGroup := r.Group(nil)
		internalRoutesGroup.Use(middlewares.CheckSubnet)
		internalRoutesGroup.Get("/stats", getStatsHandler.ServeHTTP)
	})

	router.Mount("/debug", middleware.Profiler())
	return router
}

// Run is a function that prepares all the infrastructure dependencies and settings and runs the web server.
func Run(addr string) error {
	logger.Log.Infof("starting server at %s", addr)
	doneChan := make(chan struct{})
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT|syscall.SIGTERM|syscall.SIGQUIT)
	if config.Settings.DatabaseDSN != "" {
		var err error
		Pool, err = sql.Open("pgx", config.Settings.DatabaseDSN)
		Pool.SetMaxOpenConns(config.Settings.DatabaseMaxConnections)
		if err != nil {
			return err
		}
		defer func(Pool *sql.DB) {
			closeErr := Pool.Close()
			if closeErr != nil {
				panic(closeErr)
			}
		}(Pool)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err = Pool.PingContext(ctx); err != nil {
			return err
		}
		err = migrateDB(Pool)
		if err != nil {
			return err
		}
	} else {
		err := prefillMemory()
		if err != nil {
			return err
		}
		logger.Log.Info("Memory prefilled from file")
		err = storage.FSWrapper.Open()
		if err != nil {
			return err
		}
		defer func(FSWrapper *storage.FileWrapper) {
			closeErr := FSWrapper.Close()
			if closeErr != nil {
				panic(closeErr)
			}
		}(storage.FSWrapper)
	}
	server := &http.Server{Addr: addr, Handler: ShortenURLRouter(Pool, doneChan)}
	go func() {
		<-sigint
		logger.Log.Info("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(doneChan)
	}()
	logger.Log.Info("Server initiation completed, starting to serve")
	if config.Settings.TLSEnabled {
		if err := server.ListenAndServeTLS(config.Settings.CertPath, config.Settings.KeyPath); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
	<-doneChan
	if Pool != nil {
		logger.Log.Info("shutting down db pool")
		err := Pool.Close()
		if err != nil {
			return err
		}
	}
	logger.Log.Info("Bye")
	return nil
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
		shortURLService = service.NewService(storage.MemoryRepo{}, make(chan struct{}))
		fillingError := shortURLService.FillRow(context.Background(), row.OriginalURL, row.ShortURL, row.UserID)
		if fillingError != nil {
			return fillingError
		}
	}
	return nil
}

func migrateDB(pool *sql.DB) error {
	return goose.Up(pool, "internal/app/storage/migrations")
}
