package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return newTextLogger(&bytes.Buffer{})
}

func newTextLogger(buffer *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buffer, nil))
}

func gzipRequestBody(t *testing.T, request *http.Request) {
	t.Helper()

	var body bytes.Buffer
	gzipWriter := gzip.NewWriter(&body)
	_, err := io.Copy(gzipWriter, request.Body)
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	request.Body = io.NopCloser(&body)
	request.ContentLength = int64(body.Len())
	request.Header.Set("Content-Encoding", "gzip")
}

func gunzipResponseBody(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()

	gzipReader, err := gzip.NewReader(response.Body)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, gzipReader.Close())
	}()

	body, err := io.ReadAll(gzipReader)
	require.NoError(t, err)

	return string(body)
}
