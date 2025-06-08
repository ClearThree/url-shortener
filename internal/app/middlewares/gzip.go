package middlewares

import (
	"net/http"
	"strings"

	"github.com/clearthree/url-shortener/internal/app/compress"
)

func GzipMiddleware(next http.Handler) http.Handler {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		usedWriter := writer

		acceptEncoding := request.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			compressWriter := compress.NewCompressWriter(writer)
			usedWriter = compressWriter

			defer func(compressWriter *compress.CompressWriter) {
				err := compressWriter.Close()
				if err != nil {
					panic(err)
				}
			}(compressWriter)
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
			defer func(compressReader *compress.CompressReader) {
				closeErr := compressReader.Close()
				if closeErr != nil {
					panic(closeErr)
				}
			}(compressReader)
		}

		next.ServeHTTP(usedWriter, request)
	}
	return http.HandlerFunc(fn)
}
