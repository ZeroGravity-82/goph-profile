package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildAvatarThumbnails_DecodesPNG проверяет обработку PNG-файла.
func TestBuildAvatarThumbnails_DecodesPNG(t *testing.T) {
	// Arrange
	content := testPNG(t, 320, 240)

	// Act
	thumbnails, err := BuildAvatarThumbnails(content)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 320, thumbnails.Width)
	assert.Equal(t, 240, thumbnails.Height)
	assertPNGSize(t, thumbnails.Thumb100, 100, 100)
	assertPNGSize(t, thumbnails.Thumb300, 300, 300)
}

// TestBuildAvatarThumbnails_DecodesJPEG проверяет обработку JPEG-файла.
func TestBuildAvatarThumbnails_DecodesJPEG(t *testing.T) {
	// Arrange
	content := testJPEG(t, 320, 240)

	// Act
	thumbnails, err := BuildAvatarThumbnails(content)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 320, thumbnails.Width)
	assert.Equal(t, 240, thumbnails.Height)
	assertPNGSize(t, thumbnails.Thumb100, 100, 100)
	assertPNGSize(t, thumbnails.Thumb300, 300, 300)
}

// testPNG создает PNG-файл заданного размера.
func testPNG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := testImage(width, height)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// testJPEG создает JPEG-файл заданного размера.
func testJPEG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := testImage(width, height)
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// testImage создает тестовое изображение заданного размера.
func testImage(width int, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 180, A: 255})
		}
	}
	return img
}

// assertPNGSize проверяет размеры PNG без декодирования всех пикселей.
func assertPNGSize(t *testing.T, content []byte, width int, height int) {
	t.Helper()

	cfg, err := png.DecodeConfig(bytes.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, width, cfg.Width)
	assert.Equal(t, height, cfg.Height)
}

// TestBuildAvatarThumbnails_ReturnsDecodeError проверяет ошибку декодирования файла.
func TestBuildAvatarThumbnails_ReturnsDecodeError(t *testing.T) {
	// Arrange
	content := []byte("not an image")

	// Act
	thumbnails, err := BuildAvatarThumbnails(content)

	// Assert
	require.Error(t, err)
	assert.Zero(t, thumbnails)
}
