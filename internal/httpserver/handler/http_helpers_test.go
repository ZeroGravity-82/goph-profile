package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func newResolveUserRequest(t *testing.T, email string) *http.Request {
	t.Helper()

	return newResolveUserRequestToPath(t, userResolveRoutePath, email)
}

func newResolveUserRequestToPath(t *testing.T, path string, email string) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	require.NoError(t, json.NewEncoder(body).Encode(dto.ResolveUserRequest{Email: email}))

	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newSelectCurrentAvatarRequest(
	t *testing.T,
	headerUserID string,
	avatarID string,
) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	require.NoError(t, json.NewEncoder(body).Encode(dto.SelectCurrentAvatarRequest{AvatarID: avatarID}))

	return newSelectCurrentAvatarRequestWithBody(t, headerUserID, body)
}

func newSelectCurrentAvatarRequestWithBody(
	t *testing.T,
	headerUserID string,
	body *bytes.Buffer,
) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodPatch, publicAvatarRoutePath, body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", headerUserID)

	return request
}

func newDeleteCurrentAvatarRequest(userID string) *http.Request {
	request := httptest.NewRequest(http.MethodDelete, publicAvatarRoutePath, nil)
	request.Header.Set("X-User-ID", userID)

	return request
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

func mustPublicAvatarURL(t *testing.T, email string) string {
	t.Helper()

	values := url.Values{}
	values.Set("email", email)

	return publicAvatarRoutePath + "?" + values.Encode()
}

func newUploadAvatarRequest(
	t *testing.T,
	userID string,
	fileName string,
	content []byte,
) *http.Request {
	t.Helper()

	return newUploadAvatarRequestToPath(t, avatarRoutePath, userID, fileName, content)
}

func newUploadAvatarRequestToPath(
	t *testing.T,
	path string,
	userID string,
	fileName string,
	content []byte,
) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile(formFileField, fileName)
	require.NoError(t, err)
	_, err = file.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-User-ID", userID)

	return request
}

func newUploadAvatarRequestWithoutFile(t *testing.T, userID string) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("ignored", "value"))
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-User-ID", userID)

	return request
}

func newGetAvatarMetadataRequest(t *testing.T, avatarID string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, mustAvatarMetadataURL(t, avatarID), nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(avatarIDRouteParam, avatarID)

	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func newGetAvatarRequest(t *testing.T, avatarID string, size string, format string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, mustAvatarURL(t, avatarID), nil)
	values := request.URL.Query()
	if size != "" {
		values.Set(avatarSizeQueryParam, size)
	}
	if format != "" {
		values.Set(avatarFormatQueryParam, format)
	}
	request.URL.RawQuery = values.Encode()
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(avatarIDRouteParam, avatarID)

	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func pngContent() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47,
		0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52,
	}
}

func mustAvatarURL(t *testing.T, avatarID string) string {
	t.Helper()

	avatarURL, err := url.JoinPath(avatarRoutePath, avatarID)
	require.NoError(t, err)

	return avatarURL
}

func mustAvatarThumbnailURL(t *testing.T, avatarID uuid.UUID, size string) string {
	t.Helper()

	thumbnailURL, err := avatarURLForID(avatarID)
	require.NoError(t, err)
	values := url.Values{}
	values.Set("size", size)

	return thumbnailURL + "?" + values.Encode()
}
