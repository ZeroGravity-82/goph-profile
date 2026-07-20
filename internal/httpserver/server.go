package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/handler"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

type avatarUseCase interface {
	UploadAvatar(ctx context.Context, in usecase.UploadAvatarInput) (usecase.UploadAvatarOutput, error)
	GetCurrentAvatarByEmail(
		ctx context.Context,
		in usecase.GetCurrentAvatarByEmailInput,
	) (usecase.GetCurrentAvatarByEmailOutput, error)
	GetAvatarMetadata(
		ctx context.Context,
		in usecase.GetAvatarMetadataInput,
	) (usecase.GetAvatarMetadataOutput, error)
}

type userUseCase interface {
	ResolveUserByEmail(ctx context.Context, email model.Email) (usecase.ResolveUserByEmailOutput, error)
}

// HTTPServer запускает основной REST API.
//
// Он запускает роутер, собранный handler.NewRouter, на указанном адресе.
type HTTPServer struct {
	addr          string
	avatarUseCase avatarUseCase
	userUseCase   userUseCase
	logger        *slog.Logger
}

// NewHTTPServer создает новый HTTPServer.
func NewHTTPServer(
	addr string,
	avatarUseCase avatarUseCase,
	userUseCase userUseCase,
	logger *slog.Logger,
) (*HTTPServer, error) {
	if addr == "" {
		return nil, errors.New("http server address is not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &HTTPServer{
		addr:          addr,
		avatarUseCase: avatarUseCase,
		userUseCase:   userUseCase,
		logger:        logger,
	}, nil
}

// Run запускает HTTP-сервер и блокируется, пока не отменен контекст или сервер не остановится с ошибкой.
func (s *HTTPServer) Run(ctx context.Context) error {
	router := handler.NewRouter(s.avatarUseCase, s.userUseCase, s.logger)
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
