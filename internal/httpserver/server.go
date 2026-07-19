package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/handler"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// HTTPServer запускает основной REST API.
//
// Он запускает роутер, собранный handler.NewRouter, на указанном адресе.
type HTTPServer struct {
	addr     string
	uploader handler.AvatarUploader
	logger   *slog.Logger
}

// NewHTTPServer создает HTTPServer.
func NewHTTPServer(addr string, uploader handler.AvatarUploader, logger *slog.Logger) *HTTPServer {
	if logger == nil {
		logger = logging.NopLogger()
	}
	return &HTTPServer{
		addr:     addr,
		uploader: uploader,
		logger:   logger,
	}
}

// Run запускает HTTP-сервер и блокируется, пока не отменен контекст или сервер не остановится с ошибкой.
func (s *HTTPServer) Run(ctx context.Context) error {
	router := handler.NewRouter(s.uploader, s.logger)
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("starting http server", slog.String("addr", s.addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)
		if err == nil {
			s.logger.Info("http server stopped with graceful shutdown")
			return nil
		}
		return fmt.Errorf("http server stopped with error: %w", err)
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			s.logger.Info("http server closed")
			return nil
		}
		return fmt.Errorf("http server error: %w", err)
	}
}
