package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPServer_Run_ReturnsListenError проверяет ошибку запуска HTTP-сервера.
func TestHTTPServer_Run_ReturnsListenError(t *testing.T) {
	// Arrange
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	server, err := NewHTTPServer(listener.Addr().String(), nil, discardLogger())
	require.NoError(t, err)

	// Act
	err = server.Run(context.Background())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http server error")
}

// TestHTTPServer_Run_ShutsDownOnContextCancel проверяет graceful shutdown.
func TestHTTPServer_Run_ShutsDownOnContextCancel(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewHTTPServer("127.0.0.1:0", nil, discardLogger())
	require.NoError(t, err)

	// Act
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()
	cancel()

	// Assert
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("http server did not stop after context cancellation")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
