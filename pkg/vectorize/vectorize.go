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

// Configuration for Vectorize; hard-code for now, but will need to expose these somehow.
// const backgroundColor = color.White
const maxRunLength = 20

func Vectorize(img *ColorImage) string {
	return "not implemented"
}

func FindRuns(img *ColorImage, reportRun func(center float32, width int)) {
	runStart := -1
	checkReportRun := func(x int) {
		if runStart < 0 {
			return
		}
		runLength := x - runStart
		if runLength <= maxRunLength {
			reportRun(float32(x+runStart)/2, runLength)
		}
		// End the current run.
		runStart = -1
	}

	// First pass: scan for horizontal runs, which are then assembled into vertical (or near vertical) lines.
	i := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			c := img.Data[i]
			i++
			if c == color.Black {
				if runStart == -1 {
					// new run
					runStart = x
				}
			} else {
				// Non-black; check for finished run
				checkReportRun(x)
			}
		}
		// check for finished run at end of row
		checkReportRun(img.Width)
	}
}

/* first pass at this function:
{
	// To start with, just look for white and black pixels.
	// This will of course need to be expanded to other colors, which could be done
	// trivially by running multiple passes of this alg, one for each color. But it
	// is probably more efficient to look for all colors at the same time.

	// First pass: scan for horizontal runs, which are then assembled into vertical (or near vertical) lines.
	i := 0
	runStart := -1
	//for y := 0; y < image.Height; y++ {
	// For testing, load just the middle 2/3 of the file - works especially well with letter sized pages
	//lines := [][]image.Point{}
	//priorRunCenters := []float32{}
	for y := img.Height / 6; y < img.Height*5/6; y++ {
		runCenters := []float32{}
		for x := 0; x < img.Width; x++ {
			c := img.Data[i]
			i++
			if c == color.Black {
				if runStart == -1 {
					// new run
					runStart = x
				}
			} else {
				// Non-black; check for finished run
				if runStart >= 0 {
					runLength := x - runStart
					if runLength <= maxRunLength {
						//fmt.Printf("Got a run from %d to %d, y=%d\n", runStart, x-1, y)
						runCenters = append(runCenters, float32(x+runStart))
					}

					// End the current run.
					runStart = -1
				}
			}
		}

		//priorRunCenters = runCenters
	}

	return "done"
}
*/
