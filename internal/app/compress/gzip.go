// Package compress implements Writer and Reader to handle gzip-compressed requests and compress responses.
package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// CompressWriter is a structure to contain original http.Writer along with gzip.Writer.
// Implements the http.ResponseWriter interface.
type CompressWriter struct {
	writer     http.ResponseWriter
	gzipWriter *gzip.Writer
}

// NewCompressWriter initializes the new CompressWriter object using the default http.ResponseWriter as an input.
func NewCompressWriter(writer http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		writer:     writer,
		gzipWriter: gzip.NewWriter(writer),
	}
}

// Header method returns the header map that will be sent to client in the response.
// Returns the header map from original writer just to comply the interface.
func (c *CompressWriter) Header() http.Header {
	return c.writer.Header()
}

// Write writes the response body using just the original Writer, but compresses the payload if applicable.
// If the compression is applicable, adds the Content-Encoding header to the response.
func (c *CompressWriter) Write(p []byte) (int, error) {
	if c.ShouldCompress() {
		c.writer.Header().Set("Content-Encoding", "gzip")
		return c.gzipWriter.Write(p)
	}
	return c.writer.Write(p)
}

// WriteHeader writes the response status code using just the original Writer, but if the compression is applicable,
// adds the Content-Encoding header to the response.
func (c *CompressWriter) WriteHeader(statusCode int) {
	if c.ShouldCompress() {
		c.writer.Header().Set("Content-Encoding", "gzip")
	}
	c.writer.WriteHeader(statusCode)
}

// ShouldCompress checks if response payload content-type is applicable for compression.
func (c *CompressWriter) ShouldCompress() bool {
	contentType := c.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		return true
	}
	return false
}

// Close closes the gzipWriter if it was applicable.
func (c *CompressWriter) Close() error {
	if c.ShouldCompress() {
		return c.gzipWriter.Close()
	}
	return nil
}

// CompressReader is a structure to contain original io.ReadCloser along with gzip.Reader.
// Implements the io.ReadCloser interface.
type CompressReader struct {
	reader     io.ReadCloser
	gzipReader *gzip.Reader
}

// NewCompressReader initializes the new CompressReader object using the default io.ReadCloser as an input.
func NewCompressReader(reader io.ReadCloser) (*CompressReader, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}

	return &CompressReader{
		reader:     reader,
		gzipReader: gzipReader,
	}, nil
}

// Read reads the compressed input and decompresses it.
func (c *CompressReader) Read(p []byte) (n int, err error) {
	return c.gzipReader.Read(p)
}

// Close closes both original and gzip readers.
func (c *CompressReader) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}
	return c.gzipReader.Close()
}
