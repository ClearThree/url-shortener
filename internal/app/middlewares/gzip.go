package middlewares

import (
	"github.com/clearthree/url-shortener/internal/app/compress"
	"net/http"
	"strings"
)

func GzipMiddleware(next http.Handler) http.Handler {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		usedWriter := writer

		acceptEncoding := request.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			compressWriter := compress.NewCompressWriter(writer)
			usedWriter = compressWriter

			defer compressWriter.Close()
		}

		contentEncoding := request.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			compressReader, err := compress.NewCompressReader(request.Body)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			request.Body = compressReader
			defer compressReader.Close()
		}

		next.ServeHTTP(usedWriter, request)
	}
	return http.HandlerFunc(fn)
}
