package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

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

func mustAvatarMetadataURL(t *testing.T, avatarID string) string {
	t.Helper()

	metadataURL, err := url.JoinPath(avatarRoutePath, avatarID, "metadata")
	require.NoError(t, err)

	return metadataURL
}
