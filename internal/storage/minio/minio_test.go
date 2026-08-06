package minio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	minioV7 "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMinIOStorage_RequiresConfig проверяет валидацию обязательных параметров MinIO до подключения.
func TestNewMinIOStorage_RequiresConfig(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		accessKey string
		secretKey string
		bucket    string
		wantErr   string
	}{
		{
			name:      "endpoint",
			accessKey: "access-key",
			secretKey: "secret-key",
			bucket:    "avatars",
			wantErr:   "minio endpoint is not provided",
		},
		{
			name:      "access key",
			endpoint:  "localhost:9000",
			secretKey: "secret-key",
			bucket:    "avatars",
			wantErr:   "minio access key is not provided",
		},
		{
			name:      "secret key",
			endpoint:  "localhost:9000",
			accessKey: "access-key",
			bucket:    "avatars",
			wantErr:   "minio secret key is not provided",
		},
		{
			name:      "bucket",
			endpoint:  "localhost:9000",
			accessKey: "access-key",
			secretKey: "secret-key",
			wantErr:   "minio bucket is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			storage, err := NewMinIOStorage(
				ctx,
				tt.endpoint,
				tt.accessKey,
				tt.secretKey,
				tt.bucket,
				false,
			)

			// Assert
			require.EqualError(t, err, tt.wantErr)
			assert.Nil(t, storage)
		})
	}
}

// TestMinIOStorage_ObjectKeyOriginal проверяет стабильный формат ключа исходного файла аватарки.
func TestMinIOStorage_ObjectKeyOriginal(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}
	userID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001")
	avatarID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")

	// Act
	key := storage.ObjectKeyOriginal(userID, avatarID)

	// Assert
	assert.Equal(
		t,
		"users/018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001/avatars/018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003/original",
		key,
	)
}

// TestMinIOStorage_ObjectKeyThumbnails проверяет стабильный формат ключей миниатюр аватарки.
func TestMinIOStorage_ObjectKeyThumbnails(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}
	userID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001")
	avatarID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")

	// Act
	thumb100Key := storage.ObjectKeyThumb100(userID, avatarID)
	thumb300Key := storage.ObjectKeyThumb300(userID, avatarID)

	// Assert
	assert.Equal(
		t,
		"users/018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001/avatars/018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003/thumb-100.png",
		thumb100Key,
	)
	assert.Equal(
		t,
		"users/018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001/avatars/018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003/thumb-300.png",
		thumb300Key,
	)
}

// TestMinIOStorage_PutRequiresObjectKey проверяет локальную валидацию ключа объекта перед записью.
func TestMinIOStorage_PutRequiresObjectKey(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}

	// Act
	err := storage.Put(context.Background(), "", []byte("content"))

	// Assert
	require.EqualError(t, err, "object key is not provided")
}

// TestMinIOStorage_Put_ReturnsDeadlineExceededWhenRequestStalls проверяет ограничение времени записи объекта в MinIO.
func TestMinIOStorage_Put_ReturnsDeadlineExceededWhenRequestStalls(t *testing.T) {
	// Arrange
	storage := newStalledMinIOStorage(t)

	// Act
	err := storage.Put(context.Background(), "avatar", []byte("content"))

	// Assert
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestMinIOStorage_GetRequiresObjectKey проверяет локальную валидацию ключа объекта перед чтением.
func TestMinIOStorage_GetRequiresObjectKey(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}

	// Act
	content, err := storage.Get(context.Background(), "")

	// Assert
	require.EqualError(t, err, "object key is not provided")
	assert.Nil(t, content)
}

// TestMinIOStorage_Get_ReturnsDeadlineExceededWhenReadingStalls проверяет ограничение времени чтения объекта из MinIO.
func TestMinIOStorage_Get_ReturnsDeadlineExceededWhenReadingStalls(t *testing.T) {
	// Arrange
	storage := newStalledMinIOStorage(t)

	// Act
	content, err := storage.Get(context.Background(), "avatar")

	// Assert
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, content)
}

// TestMinIOStorage_DeleteRequiresObjectKey проверяет локальную валидацию ключа объекта перед удалением.
func TestMinIOStorage_DeleteRequiresObjectKey(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}

	// Act
	err := storage.Delete(context.Background(), "")

	// Assert
	require.EqualError(t, err, "object key is not provided")
}

// TestMinIOStorage_Delete_ReturnsDeadlineExceededWhenRequestStalls проверяет ограничение времени удаления объекта из
// MinIO.
func TestMinIOStorage_Delete_ReturnsDeadlineExceededWhenRequestStalls(t *testing.T) {
	// Arrange
	storage := newStalledMinIOStorage(t)

	// Act
	err := storage.Delete(context.Background(), "avatar")

	// Assert
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func newStalledMinIOStorage(t *testing.T) *MinIOStorage {
	t.Helper()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Length", "1")
			w.Header().Set("ETag", `"test-etag"`)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})
	client, err := minioV7.New(strings.TrimPrefix(server.URL, "http://"), &minioV7.Options{
		Creds:  credentials.NewStaticV4("access-key", "secret-key", ""),
		Region: "us-east-1",
	})
	require.NoError(t, err)
	return &MinIOStorage{
		client:           client,
		bucket:           "avatars",
		breaker:          newMinIOCircuitBreaker(minioCircuitBreakerFailureThreshold, time.Minute),
		operationTimeout: 20 * time.Millisecond,
	}
}
