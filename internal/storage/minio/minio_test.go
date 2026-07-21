package minio

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

// TestMinIOStorage_PutRequiresObjectKey проверяет локальную валидацию ключа объекта перед записью.
func TestMinIOStorage_PutRequiresObjectKey(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}

	// Act
	err := storage.Put(context.Background(), "", []byte("content"))

	// Assert
	require.EqualError(t, err, "object key is not provided")
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

// TestMinIOStorage_DeleteRequiresObjectKey проверяет локальную валидацию ключа объекта перед удалением.
func TestMinIOStorage_DeleteRequiresObjectKey(t *testing.T) {
	// Arrange
	storage := &MinIOStorage{}

	// Act
	err := storage.Delete(context.Background(), "")

	// Assert
	require.EqualError(t, err, "object key is not provided")
}
