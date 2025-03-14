package logger

import (
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	log, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = log.Sugar()
	return nil
}

func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		duration := time.Since(start)

		next.ServeHTTP(ww, r)

		Log.Infoln(
			"Processed request",
			"uri", r.RequestURI,
			"method", r.Method,
			"status", ww.Status(),
			"duration", duration,
			"size", ww.BytesWritten(),
		)
	}
	return http.HandlerFunc(fn)
}

func init() {
	err := Initialize(config.Settings.LogLevel)
	if err != nil {
		log.Fatal("error initializing logger")
	}
}
