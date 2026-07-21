package imageproc

import (
	"bytes"
	stdimage "image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildAvatarThumbnails проверяет создание PNG-миниатюр аватарки и сохранение исходных размеров.
func TestBuildAvatarThumbnails(t *testing.T) {
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

// testPNG создает PNG-изображение заданного размера для проверки обработки аватарок.
func testPNG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 180, A: 255})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// assertPNGSize проверяет размеры PNG-изображения без декодирования всех пикселей.
func assertPNGSize(t *testing.T, content []byte, width int, height int) {
	t.Helper()

	cfg, err := png.DecodeConfig(bytes.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, width, cfg.Width)
	assert.Equal(t, height, cfg.Height)
}

// TestBuildAvatarThumbnails_ReturnsDecodeError проверяет ошибку декодирования исходного файла.
func TestBuildAvatarThumbnails_ReturnsDecodeError(t *testing.T) {
	// Arrange
	content := []byte("not an image")

	// Act
	thumbnails, err := BuildAvatarThumbnails(content)

	// Assert
	require.Error(t, err)
	assert.Zero(t, thumbnails)
}
