package utilities

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
)

func GenerateTrackingPixel() bytes.Buffer {
	width := 1
	height := 1

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	// Colors are defined by Red, Green, Blue, Alpha uint8 values.
	alpha := color.RGBA{255, 255, 255, 0x00}
	img.Set(0, 0, alpha)

	// Encode as PNG.
	var b bytes.Buffer
	foo := io.Writer(&b)
	png.Encode(foo, img)
	return b
}
