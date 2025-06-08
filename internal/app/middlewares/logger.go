package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/clearthree/url-shortener/internal/app/logger"
)

func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		duration := time.Since(start)
		logger.Log.Infoln(
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
