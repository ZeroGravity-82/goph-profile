package httpserver

import (
	"context"
	"crypto/tls"
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
	SelectCurrentAvatar(
		ctx context.Context,
		in usecase.SelectCurrentAvatarInput,
	) error
	DeleteCurrentAvatar(ctx context.Context, in usecase.DeleteCurrentAvatarInput) error
	DeleteAvatar(ctx context.Context, in usecase.DeleteAvatarInput) error
	ListUserAvatars(
		ctx context.Context,
		in usecase.ListUserAvatarsInput,
	) (usecase.ListUserAvatarsOutput, error)
	GetCurrentAvatarByEmail(
		ctx context.Context,
		in usecase.GetCurrentAvatarByEmailInput,
	) (usecase.GetCurrentAvatarByEmailOutput, error)
	GetCurrentAvatarByUserID(
		ctx context.Context,
		in usecase.GetCurrentAvatarByUserIDInput,
	) (usecase.GetCurrentAvatarByUserIDOutput, error)
	GetAvatar(ctx context.Context, in usecase.GetAvatarInput) (usecase.GetAvatarOutput, error)
	GetAvatarMetadata(
		ctx context.Context,
		in usecase.GetAvatarMetadataInput,
	) (usecase.GetAvatarMetadataOutput, error)
}

type userUseCase interface {
	ResolveUserByEmail(ctx context.Context, email model.Email) (usecase.ResolveUserByEmailOutput, error)
}

// HealthCheck проверяет доступность внешней зависимости.
type HealthCheck func(ctx context.Context) error

// HealthChecks содержит именованные проверки внешних зависимостей.
type HealthChecks map[string]HealthCheck

func (checks HealthChecks) handlerChecks() map[string]func(context.Context) error {
	result := make(map[string]func(context.Context) error, len(checks))
	for name, check := range checks {
		result[name] = check
	}
	return result
}

// HTTPServer запускает основной REST API.
//
// Он запускает роутер, собранный handler.NewRouter, на указанном адресе.
type HTTPServer struct {
	addr          string
	tlsConfig     *tls.Config
	avatarUseCase avatarUseCase
	userUseCase   userUseCase
	healthChecks  HealthChecks
	logger        *slog.Logger
}

// NewHTTPServer создает HTTPServer.
func NewHTTPServer(
	addr string,
	tlsConfig *tls.Config,
	avatarUseCase avatarUseCase,
	userUseCase userUseCase,
	healthChecks HealthChecks,
	logger *slog.Logger,
) (*HTTPServer, error) {
	if addr == "" {
		return nil, errors.New("http server address is not provided")
	}
	if tlsConfig == nil {
		return nil, errors.New("TLS config is not provided")
	}
	if avatarUseCase == nil {
		return nil, errors.New("avatar usecase is not provided")
	}
	if userUseCase == nil {
		return nil, errors.New("user usecase is not provided")
	}
	if len(healthChecks) == 0 {
		return nil, errors.New("health checks are not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &HTTPServer{
		addr:          addr,
		tlsConfig:     tlsConfig,
		avatarUseCase: avatarUseCase,
		userUseCase:   userUseCase,
		healthChecks:  healthChecks,
		logger:        logger,
	}, nil
}

// Run запускает HTTP-сервер и блокируется, пока не отменен контекст или сервер не остановится с ошибкой.
func (s *HTTPServer) Run(ctx context.Context) error {
	logger := s.logger.With("component", "httpserver")
	router, err := handler.NewRouter(
		s.avatarUseCase,
		s.userUseCase,
		s.healthChecks.handlerChecks(),
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		TLSConfig:         s.tlsConfig,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "starting http server", slog.String("addr", s.addr))
		errCh <- srv.ListenAndServeTLS("", "")
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)
		if err == nil {
			logger.InfoContext(shutdownCtx, "http server stopped with graceful shutdown")
			return nil
		}
		return fmt.Errorf("failed to stop http server: %w", err)
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			logger.InfoContext(ctx, "http server closed")
			return nil
		}
		return fmt.Errorf("failed to run http server: %w", err)
	}
}
