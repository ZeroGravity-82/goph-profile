package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
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

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, wantError string) {
	t.Helper()

	var body dto.ErrorResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, wantError, body.Error)
}

func mustAvatarMetadataURL(t *testing.T, avatarID string) string {
	t.Helper()

	metadataURL, err := url.JoinPath(avatarRoutePath, avatarID, "metadata")
	require.NoError(t, err)

	return metadataURL
}
