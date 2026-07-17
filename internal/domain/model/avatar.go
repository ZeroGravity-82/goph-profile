package model

import "time"

const (
	// MaxFileNameLength ограничивает исходное имя файла аватарки.
	MaxFileNameLength = 255
	// MaxMIMETypeLength ограничивает MIME-тип исходного файла.
	MaxMIMETypeLength = 255
	// MaxObjectKeyLength ограничивает ключ файла аватарки.
	MaxObjectKeyLength = 512
	// MaxAvatarFileSizeBytes ограничивает размер исходного файла 10 МиБ.
	MaxAvatarFileSizeBytes int64 = 10 * 1024 * 1024
	// MaxImageWidth ограничивает ширину изображения в пикселях.
	MaxImageWidth = 4096
	// MaxImageHeight ограничивает высоту изображения в пикселях.
	MaxImageHeight = 4096
	// MaxImagePixels ограничивает число пикселей изображения.
	MaxImagePixels = 16_777_216
)

const (
	// MIMEJPEG содержит MIME-тип JPEG.
	MIMEJPEG = "image/jpeg"
	// MIMEPNG содержит MIME-тип PNG.
	MIMEPNG = "image/png"
	// MIMEWebP содержит MIME-тип WebP.
	MIMEWebP = "image/webp"
)

// AvatarStatus описывает состояние жизненного цикла аватарки.
type AvatarStatus string

const (
	// AvatarStatusProcessing означает обработку.
	AvatarStatusProcessing AvatarStatus = "processing"
	// AvatarStatusReady означает готовность.
	AvatarStatusReady AvatarStatus = "ready"
	// AvatarStatusFailed означает, что обработка завершилась ошибкой.
	AvatarStatusFailed AvatarStatus = "failed"
	// AvatarStatusDeleting означает удаление.
	AvatarStatusDeleting AvatarStatus = "deleting"
	// AvatarStatusDeleted означает, что файлы аватарки удалены.
	AvatarStatusDeleted AvatarStatus = "deleted"
)

// Avatar описывает аватарку пользователя и ключи ее файлов.
type Avatar struct {
	ID                AvatarID
	UserID            UserID
	FileName          string
	MIMEType          string
	SizeBytes         int64
	Width             *int
	Height            *int
	ObjectKeyOriginal string
	ObjectKeyThumb100 *string
	ObjectKeyThumb300 *string
	Status            AvatarStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// NewProcessingAvatar создает аватарку после загрузки исходного файла.
func NewProcessingAvatar(
	id AvatarID,
	userID UserID,
	fileName string,
	mimeType string,
	sizeBytes int64,
	objectKeyOriginal string,
	now time.Time,
) (Avatar, error) {
	if err := validateAvatarID(id); err != nil {
		return Avatar{}, err
	}
	if err := validateUserID(userID); err != nil {
		return Avatar{}, err
	}
	if err := validateAvatarMetadata(fileName, mimeType, sizeBytes, objectKeyOriginal); err != nil {
		return Avatar{}, err
	}
	return Avatar{
		ID:                id,
		UserID:            userID,
		FileName:          fileName,
		MIMEType:          mimeType,
		SizeBytes:         sizeBytes,
		ObjectKeyOriginal: objectKeyOriginal,
		Status:            AvatarStatusProcessing,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// CanBeCurrent проверяет, можно ли выбрать аватарку текущей.
func (a Avatar) CanBeCurrent() error {
	if a.DeletedAt != nil || a.Status == AvatarStatusDeleting || a.Status == AvatarStatusDeleted {
		return ErrAvatarDeleted
	}
	if a.Status != AvatarStatusReady {
		return ErrAvatarNotReady
	}
	return nil
}

// MarkReady помечает аватарку готовой.
func (a *Avatar) MarkReady(
	width int,
	height int,
	objectKeyThumb100 string,
	objectKeyThumb300 string,
	now time.Time,
) error {
	if a.Status != AvatarStatusProcessing {
		return ErrInvalidAvatarTransition
	}
	if err := ValidateImageDimensions(width, height); err != nil {
		return err
	}
	if !validLength(objectKeyThumb100, MaxObjectKeyLength) ||
		!validLength(objectKeyThumb300, MaxObjectKeyLength) {
		return ErrInvalidAvatarMetadata
	}
	a.Width = intPtr(width)
	a.Height = intPtr(height)
	a.ObjectKeyThumb100 = stringPtr(objectKeyThumb100)
	a.ObjectKeyThumb300 = stringPtr(objectKeyThumb300)
	a.Status = AvatarStatusReady
	a.UpdatedAt = now
	return nil
}

// MarkFailed переводит аватарку в состояние ошибки обработки.
func (a *Avatar) MarkFailed(now time.Time) error {
	if a.Status != AvatarStatusProcessing {
		return ErrInvalidAvatarTransition
	}
	a.Status = AvatarStatusFailed
	a.UpdatedAt = now
	return nil
}

// MarkDeleting выполняет мягкое удаление аватарки.
func (a *Avatar) MarkDeleting(now time.Time) error {
	if a.Status == AvatarStatusDeleted {
		return nil
	}
	if a.DeletedAt != nil || a.Status == AvatarStatusDeleting {
		return nil
	}
	a.Status = AvatarStatusDeleting
	a.DeletedAt = &now
	a.UpdatedAt = now
	return nil
}

// MarkDeleted завершает физическое удаление файлов аватарки.
func (a *Avatar) MarkDeleted(now time.Time) error {
	if a.DeletedAt == nil && a.Status != AvatarStatusDeleting {
		return ErrInvalidAvatarTransition
	}
	a.Status = AvatarStatusDeleted
	a.UpdatedAt = now
	return nil
}

// ValidateImageDimensions проверяет лимиты изображения в пикселях.
func ValidateImageDimensions(width int, height int) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidAvatarMetadata
	}
	if width > MaxImageWidth || height > MaxImageHeight {
		return ErrImageTooLarge
	}
	if width*height > MaxImagePixels {
		return ErrImageTooLarge
	}
	return nil
}

func validateAvatarMetadata(fileName string, mimeType string, sizeBytes int64, objectKeyOriginal string) error {
	if !validLength(fileName, MaxFileNameLength) {
		return ErrInvalidAvatarMetadata
	}
	if !supportedMIMEType(mimeType) || !validLength(mimeType, MaxMIMETypeLength) {
		return ErrInvalidAvatarMetadata
	}
	if sizeBytes <= 0 {
		return ErrInvalidAvatarMetadata
	}
	if sizeBytes > MaxAvatarFileSizeBytes {
		return ErrFileTooLarge
	}
	if !validLength(objectKeyOriginal, MaxObjectKeyLength) {
		return ErrInvalidAvatarMetadata
	}
	return nil
}

func supportedMIMEType(mimeType string) bool {
	switch mimeType {
	case MIMEJPEG, MIMEPNG, MIMEWebP:
		return true
	default:
		return false
	}
}

func validLength(value string, maxLength int) bool {
	return value != "" && len(value) <= maxLength
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
