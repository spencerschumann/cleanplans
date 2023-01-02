package vectorize

import (
	"cleanplans/pkg/color"
	"image"
	imgcolor "image/color"
)

// Terrible name...if this works, I need to change names to avoid collisions with the standard Go image and color packages.
type ColorImage struct {
	Width  int
	Height int
	Data   []color.Color
}

func (ci *ColorImage) ColorModel() imgcolor.Model {
	return color.Palette
}

func (ci *ColorImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, ci.Width, ci.Height)
}

func (ci *ColorImage) At(x, y int) imgcolor.Color {
	return color.Palette[ci.ColorIndexAt(x, y)]
}

func (ci *ColorImage) ColorIndexAt(x, y int) uint8 {
	return uint8(ci.Data[x+y*ci.Width])
}

// PDFJSImageToColorImage converts the input image data via color.RemapColor,
// returning a slice of color.Color values with the same width and height
// as the input image.
func PDFJSImageToColorImage(image []byte, width, height, bitsPerPixel int) *ColorImage {
	if bitsPerPixel == 1 {
		// Not yet supported.
		return nil
	}

	stride := 0
	if bitsPerPixel == 32 {
		stride = 4
	}
	if bitsPerPixel == 24 {
		stride = 3
	}

	size := len(image)
	data := make([]color.Color, width*height)
	j := 0
	for i := 0; i < size; i += stride {
		// Ignore alpha for now - assume fully opaque images
		data[j] = color.RemapColor(image[i], image[i+1], image[i+2])
		j++
	}
	return &ColorImage{
		Width:  width,
		Height: height,
		Data:   data,
	}
}
