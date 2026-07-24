package imageproc

import (
	"bytes"
	"fmt"
	stdimage "image"
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	thumbnail100Size = 100
	thumbnail300Size = 300
)

// AvatarThumbnails содержит исходные размеры изображения и байты PNG-файлов миниатюр аватарки.
type AvatarThumbnails struct {
	Width    int
	Height   int
	Thumb100 []byte
	Thumb300 []byte
}

// BuildAvatarThumbnails декодирует изображение из байтов файла и создает квадратные PNG-миниатюры 100x100 и 300x300.
func BuildAvatarThumbnails(content []byte) (AvatarThumbnails, error) {
	source, _, err := stdimage.Decode(bytes.NewReader(content))
	if err != nil {
		return AvatarThumbnails{}, fmt.Errorf("failed to decode avatar image: %w", err)
	}

	bounds := source.Bounds()
	thumb100, err := buildSquarePNGThumbnail(source, thumbnail100Size)
	if err != nil {
		return AvatarThumbnails{}, err
	}
	thumb300, err := buildSquarePNGThumbnail(source, thumbnail300Size)
	if err != nil {
		return AvatarThumbnails{}, err
	}

	return AvatarThumbnails{
		Width:    bounds.Dx(),
		Height:   bounds.Dy(),
		Thumb100: thumb100,
		Thumb300: thumb300,
	}, nil
}

// buildSquarePNGThumbnail вырезает центр изображения, масштабирует его до квадрата и кодирует результат в PNG.
func buildSquarePNGThumbnail(source stdimage.Image, size int) ([]byte, error) {
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), source, squareCropBounds(source.Bounds()), draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("failed to encode avatar thumbnail: %w", err)
	}
	return buf.Bytes(), nil
}

// squareCropBounds возвращает квадратную область из центра прямоугольника с максимально возможной стороной.
func squareCropBounds(bounds stdimage.Rectangle) stdimage.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()
	side := min(width, height)
	x0 := bounds.Min.X + (width-side)/2
	y0 := bounds.Min.Y + (height-side)/2

	return stdimage.Rect(x0, y0, x0+side, y0+side)
}
