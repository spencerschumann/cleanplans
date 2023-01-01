package vectorize

import "cleanplans/pkg/color"

// PDFJSImageToColorImage converts the input image data via color.RemapColor,
// returning a slice of color.Color values with the same width and height
// as the input image.
func PDFJSImageToColorImage(image []byte, width, height, bitsPerPixel int) []color.Color {
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

	size := width * height
	output := make([]color.Color, size)
	j := 0
	for i := 0; i < size; i += stride {
		// Ignore alpha for now - assume fully opaque images
		output[j] = color.RemapColor(image[i], image[i+1], image[i+2])
		j++
	}
	return output
}
