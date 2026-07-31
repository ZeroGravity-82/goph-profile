package handler

import (
	"compress/gzip"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

func withGzip(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = logging.NopLogger()
	}
	logger = logger.With("component", "httpserver.gzip_middleware")

	compressResponse := middleware.Compress(gzip.DefaultCompression, "application/json")

	return func(next http.Handler) http.Handler {
		return compressResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !hasGzipEncoding(r.Header.Get("Content-Encoding")) {
				next.ServeHTTP(w, r)
				return
			}

			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				logger.WarnContext(r.Context(), "failed to create gzip request reader", slog.Any("error", err))
				writeError(logger, w, r, http.StatusBadRequest, "Invalid gzip request body", "")
				return
			}
			defer func() {
				if err = gzipReader.Close(); err != nil {
					logger.WarnContext(r.Context(), "failed to close gzip request reader", slog.Any("error", err))
				}
			}()

			r.Body = gzipReader
			r.Header.Del("Content-Encoding")
			next.ServeHTTP(w, r)
		}))
	}
}

func hasGzipEncoding(header string) bool {
	for _, encoding := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(encoding), "gzip") {
			return true
		}
	}
	return false
}
