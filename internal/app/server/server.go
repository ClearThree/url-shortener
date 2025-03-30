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
	"net/http"
	"time"
)

var Pool *sql.DB
var shortURLService = service.NewService(storage.MemoryRepo{})

func ShortenURLRouter(pool *sql.DB) chi.Router {
	var createHandler = handlers.NewCreateShortURLHandler(&shortURLService)
	var createJSONShortURLHandler = handlers.NewCreateJSONShortURLHandler(&shortURLService)
	var redirectHandler = handlers.NewRedirectToOriginalURLHandler(&shortURLService)
	var shortURLServiceDB = service.NewService(storage.NewDBRepo(pool))
	var pingHandler = handlers.NewPingHandler(&shortURLServiceDB)

	router := chi.NewRouter()
	router.Use(middlewares.RequestLogger)
	router.Use(middlewares.GzipMiddleware)
	router.Use(middleware.Recoverer)
	router.Post("/", createHandler.ServeHTTP)
	router.Post("/api/shorten", createJSONShortURLHandler.ServeHTTP)
	router.Get("/{id}", redirectHandler.ServeHTTP)
	router.Get("/ping", pingHandler.ServeHTTP)
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
	if config.Settings.DatabaseDSN != "" {
		Pool, err = sql.Open("pgx", config.Settings.DatabaseDSN)
		if err != nil {
			return err
		}
		defer Pool.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err = Pool.PingContext(ctx); err != nil {
			panic(err)
		}
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
		fillingError := shortURLService.FillRow(row.OriginalURL, row.ShortURL)
		if fillingError != nil {
			return fillingError
		}
	}
	return nil
}
