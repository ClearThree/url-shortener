package server

import (
	"context"
	"database/sql"
	"errors"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/handlers"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/middlewares"
	"github.com/clearthree/url-shortener/internal/app/service"
	"github.com/clearthree/url-shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"
	"net/http"
	"time"
)

var Pool *sql.DB
var shortURLService service.ShortURLService

func ShortenURLRouter(pool *sql.DB) chi.Router {
	if pool == nil {
		shortURLService = service.NewService(storage.MemoryRepo{})
	} else {
		shortURLService = service.NewService(storage.NewDBRepo(pool))
	}
	var createHandler = handlers.NewCreateShortURLHandler(&shortURLService)
	var createJSONShortURLHandler = handlers.NewCreateJSONShortURLHandler(&shortURLService)
	var redirectHandler = handlers.NewRedirectToOriginalURLHandler(&shortURLService)
	var shortURLServiceDB = service.NewService(storage.NewDBRepo(pool))
	var pingHandler = handlers.NewPingHandler(&shortURLServiceDB)
	var batchCreateHandler = handlers.NewBatchCreateShortURLHandler(&shortURLService)

	router := chi.NewRouter()
	router.Use(middlewares.RequestLogger)
	router.Use(middlewares.GzipMiddleware)
	router.Use(middleware.Recoverer)
	router.Post("/", createHandler.ServeHTTP)
	router.Post("/api/shorten", createJSONShortURLHandler.ServeHTTP)
	router.Post("/api/shorten/batch", batchCreateHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	router.Get("/ping", pingHandler.ServeHTTP)
	return router
}

func Run(addr string) error {
	logger.Log.Infof("starting server at %s", addr)
	if config.Settings.DatabaseDSN != "" {
		var err error
		Pool, err = sql.Open("pgx", config.Settings.DatabaseDSN)
		if err != nil {
			return err
		}
		defer Pool.Close()
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
		defer storage.FSWrapper.Close()
	}
	logger.Log.Info("Server initiation completed, starting to serve")
	return http.ListenAndServe(addr, ShortenURLRouter(Pool))
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
		shortURLService = service.NewService(storage.MemoryRepo{})
		fillingError := shortURLService.FillRow(context.Background(), row.OriginalURL, row.ShortURL)
		if fillingError != nil {
			return fillingError
		}
	}
	return nil
}

func migrateDB(pool *sql.DB) error {
	return goose.Up(pool, "internal/app/storage/migrations")
}
