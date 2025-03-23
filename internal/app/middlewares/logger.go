package middlewares

import (
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		duration := time.Since(start)

		next.ServeHTTP(ww, r)

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
