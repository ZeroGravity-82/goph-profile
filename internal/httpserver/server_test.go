package httpserver

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPServer_Run_ReturnsListenError проверяет ошибку запуска HTTP-сервера.
func TestHTTPServer_Run_ReturnsListenError(t *testing.T) {
	// Arrange
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	server, err := NewHTTPServer(
		listener.Addr().String(),
		testTLSConfig(),
		&avatarUseCaseFake{},
		&userUseCaseFake{},
		testReadinessChecks(),
		discardLogger(),
	)
	require.NoError(t, err)

	// Act
	err = server.Run(context.Background())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run http server")
}

// TestHTTPServer_Run_ShutsDownOnContextCancel проверяет graceful shutdown.
func TestHTTPServer_Run_ShutsDownOnContextCancel(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewHTTPServer(
		"127.0.0.1:0",
		testTLSConfig(),
		&avatarUseCaseFake{},
		&userUseCaseFake{},
		testReadinessChecks(),
		discardLogger(),
	)
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

// TestHTTPServer_Run_ServesHTTPWithoutTLS проверяет запуск HTTP-сервера без TLS-конфигурации.
func TestHTTPServer_Run_ServesHTTPWithoutTLS(t *testing.T) {
	// Arrange
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	ctx, cancel := context.WithCancel(context.Background())
	server, err := NewHTTPServer(
		addr,
		nil,
		&avatarUseCaseFake{},
		&userUseCaseFake{},
		testReadinessChecks(),
		discardLogger(),
	)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case runErr := <-errCh:
			require.NoError(t, runErr)
		case <-time.After(time.Second):
			t.Fatal("http server did not stop after context cancellation")
		}
	}()

	client := &http.Client{Timeout: 100 * time.Millisecond}

	// Act
	response := waitForHTTPResponse(t, client, "http://"+addr+"/live")
	defer func() { _ = response.Body.Close() }()

	// Assert
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

// TestHTTPServer_Run_ServesHTTPSWithTLS проверяет запуск HTTP-сервера с TLS-конфигурацией.
func TestHTTPServer_Run_ServesHTTPSWithTLS(t *testing.T) {
	// Arrange
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	tlsConfig, client := testTLSConfigAndClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	server, err := NewHTTPServer(
		addr,
		tlsConfig,
		&avatarUseCaseFake{},
		&userUseCaseFake{},
		testReadinessChecks(),
		discardLogger(),
	)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case runErr := <-errCh:
			require.NoError(t, runErr)
		case <-time.After(time.Second):
			t.Fatal("http server did not stop after context cancellation")
		}
	}()

	// Act
	response := waitForHTTPResponse(t, client, "https://"+addr+"/live")
	defer func() { _ = response.Body.Close() }()

	// Assert
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testTLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{
			{Certificate: [][]byte{[]byte("test certificate")}},
		},
	}
}

func testTLSConfigAndClient(t *testing.T) (*tls.Config, *http.Client) {
	t.Helper()

	fixture := httptest.NewTLSServer(http.NotFoundHandler())
	tlsConfig := fixture.TLS.Clone()
	client := fixture.Client()
	fixture.Close()
	client.Timeout = 100 * time.Millisecond
	return tlsConfig, client
}

func testReadinessChecks() ReadinessChecks {
	return ReadinessChecks{
		"test": func(_ context.Context) error {
			return nil
		},
	}
}

func waitForHTTPResponse(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	var response *http.Response
	require.Eventually(t, func() bool {
		var err error
		response, err = client.Get(url)
		return err == nil
	}, time.Second, 10*time.Millisecond)
	return response
}
