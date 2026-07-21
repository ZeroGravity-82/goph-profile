package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	minioV7 "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage хранит файлы аватарок в S3-совместимом объектном хранилище.
type MinIOStorage struct {
	client *minioV7.Client
	bucket string
}

// NewMinIOStorage создает MinIOStorage, проверяет наличие bucket и создает его при необходимости.
func NewMinIOStorage(
	ctx context.Context,
	endpoint string,
	accessKey string,
	secretKey string,
	bucket string,
	useSSL bool,
) (*MinIOStorage, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, errors.New("minio endpoint is not provided")
	}
	if strings.TrimSpace(accessKey) == "" {
		return nil, errors.New("minio access key is not provided")
	}
	if strings.TrimSpace(secretKey) == "" {
		return nil, errors.New("minio secret key is not provided")
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, errors.New("minio bucket is not provided")
	}

	client, err := minioV7.New(endpoint, &minioV7.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check minio bucket: %w", err)
	}
	if !exists {
		if err = client.MakeBucket(ctx, bucket, minioV7.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create minio bucket: %w", err)
		}
	}

	return &MinIOStorage{client: client, bucket: bucket}, nil
}

// ObjectKeyOriginal возвращает стабильный ключ исходного файла аватарки в объектном хранилище.
func (s *MinIOStorage) ObjectKeyOriginal(userID uuid.UUID, avatarID uuid.UUID) string {
	return fmt.Sprintf("users/%s/avatars/%s/original", userID, avatarID)
}

// ObjectKeyThumb100 возвращает стабильный ключ миниатюры аватарки размером 100x100.
func (s *MinIOStorage) ObjectKeyThumb100(userID uuid.UUID, avatarID uuid.UUID) string {
	return fmt.Sprintf("users/%s/avatars/%s/thumb-100.png", userID, avatarID)
}

// ObjectKeyThumb300 возвращает стабильный ключ миниатюры аватарки размером 300x300.
func (s *MinIOStorage) ObjectKeyThumb300(userID uuid.UUID, avatarID uuid.UUID) string {
	return fmt.Sprintf("users/%s/avatars/%s/thumb-300.png", userID, avatarID)
}

// Put сохраняет объект по ключу.
func (s *MinIOStorage) Put(ctx context.Context, objectKey string, content []byte) error {
	if err := validateObjectKey(objectKey); err != nil {
		return err
	}

	reader := bytes.NewReader(content)
	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		reader,
		int64(len(content)),
		minioV7.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to put minio object: %w", err)
	}
	return nil
}

func validateObjectKey(objectKey string) error {
	if strings.TrimSpace(objectKey) == "" {
		return errors.New("object key is not provided")
	}
	return nil
}

// Ping проверяет доступность S3-хранилища и bucket с файлами аватарок.
func (s *MinIOStorage) Ping(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check minio bucket: %w", err)
	}
	if !exists {
		return errors.New("minio bucket does not exist")
	}
	return nil
}

// Get читает объект по ключу.
func (s *MinIOStorage) Get(ctx context.Context, objectKey string) ([]byte, error) {
	if err := validateObjectKey(objectKey); err != nil {
		return nil, err
	}

	object, err := s.client.GetObject(ctx, s.bucket, objectKey, minioV7.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get minio object: %w", err)
	}
	defer func() {
		_ = object.Close()
	}()

	content, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read minio object: %w", err)
	}
	return content, nil
}

// Delete удаляет объект по ключу.
func (s *MinIOStorage) Delete(ctx context.Context, objectKey string) error {
	if err := validateObjectKey(objectKey); err != nil {
		return err
	}

	if err := s.client.RemoveObject(ctx, s.bucket, objectKey, minioV7.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete minio object: %w", err)
	}
	return nil
}
