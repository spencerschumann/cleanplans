package vectorize

import (
	"cleanplans/pkg/cfg"
	"cleanplans/pkg/cleaner"
	"cleanplans/pkg/color"
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"fmt"
	"image"
	imgcolor "image/color"
	"math"
	"strconv"
)

// Terrible name...if this works, I need to change names to avoid collisions with the standard Go image and color packages.
type ColorImage struct {
	Width  int
	Height int
	Data   []color.Color
}

type Point struct {
	X float32
	Y float32
}

type Line []Point

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
	data := make([]color.Color, width*height)
	j := 0
	if bitsPerPixel == 1 {
		x := 0
	readImage:
		for _, b := range image {
			for bit := 7; bit >= 0; bit-- {
				if b&(1<<bit) == 0 {
					data[j] = color.Black
				} else {
					data[j] = color.White
				}
				j++
				x++
				if x >= width {
					x = 0
					break
				}
				if j >= len(data) {
					break readImage
				}
			}
		}
	} else if bitsPerPixel == 32 || bitsPerPixel == 24 {
		stride := bitsPerPixel / 8
		size := len(image)
		for i := 0; i < size; i += stride {
			// Ignore alpha for now - assume fully opaque images
			data[j] = color.RemapColor(image[i], image[i+1], image[i+2])
			j++
		}
	} else {
		fmt.Printf("Error! bits per pixel is %d, not one of the supported values 1/24/32.\n", bitsPerPixel)
	}

	return &ColorImage{
		Width:  width,
		Height: height,
		Data:   data,
	}
}

type RunHandler interface {
	AddRun(major float32, width int)
	NextMinor()
}

// This helps close gaps, but it also doesn't take into consideration line slope. It may
// be better to do away with this when adding the code to join line segments to eliminate
// gaps.
func adjustLineEndpoints(line JoinerLine) JoinerLine {
	// Actually, let's just try trimming it down - the first and last points are dubious.
	if len(line) > 4 {
		return line[1 : len(line)-2]
	}

	last := len(line) - 1
	for i := 1; i < last; i++ {
		line[i].Minor += 0.5
	}
	line[last].Minor += 1
	return line
}

func clearHorizontalRuns(img *ColorImage, line JoinerLine) {
	for _, pt := range line {
		xStart := int(pt.Major - float32(pt.Width)/2)
		xEnd := xStart + pt.Width
		for x := xStart; x < xEnd; x++ {
			img.Data[x+int(pt.Minor)*img.Width] = color.White
		}
	}
}

func filterLines(lines []JoinerLine) []JoinerLine {
	var output []JoinerLine
	for _, line := range lines {
		output = append(output, filterLine(line)...)
	}
	return output
}

func filterLine(line JoinerLine) []JoinerLine {
	// Find median width
	counts := make([]int, cfg.VectorizeMaxRunLength+1)
	for _, pt := range line {
		counts[pt.Width]++
	}
	maxCount := 0
	median := 0
	for i, count := range counts {
		if maxCount < count {
			maxCount = count
			median = i
		}
	}

	// Only allow widths of median +/- 3 (one pixel on each side, plus one for slop) or 20%
	widthOk := func(width int) bool {
		diff := math.Abs(float64(median - width))
		return diff <= 3 || diff < (float64(median)*.2)
	}

	bestRunStart := -1
	var lines []JoinerLine
	checkReportRun := func(i int) {
		if bestRunStart < 0 {
			return
		}
		// TODO: may want to further trim the beginning and end of the subline with more strict requirements
		subLine := line[bestRunStart:i]
		if IsLineAdmissable(subLine) {
			lines = append(lines, subLine)
		}
		bestRunStart = -1
	}

	for i, pt := range line {
		if widthOk(pt.Width) {
			if bestRunStart < 0 {
				bestRunStart = i
			}
		} else {
			checkReportRun(i)
		}
	}
	checkReportRun(len(line))
	//fmt.Printf("  filterLine(%v) =>\n    %v\n", line, lines)
	return lines
}

func Vectorize(img *ColorImage) string {
	horizontalRunPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#009900;stroke-width:1;stroke-linecap:butt;stroke-linejoin:miter;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	verticalRunPathNode := &horizontalRunPathNode
	/*verticalRunPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#0000cc;stroke-width:1;stroke-linecap:butt;stroke-linejoin:miter;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}*/
	svg := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "svg"},
		Children: []*cleaner.SVGXMLNode{&horizontalRunPathNode /*&verticalRunPathNode*/},

		// Note: not using a unit specifier here for display, to match up with the png image. For
		// the final output SVG (if that is the format I go with), these will need to be mapped to mm.
		Width:  strconv.Itoa(img.Width),
		Height: strconv.Itoa(img.Height),
	}

	pj := NewPointJoiner(10, img.Width, 1)
	//fmt.Println("\n******** Find Horizontal Runs")
	FindHorizontalRuns(img, pj)
	lines := filterLines(pj.JoinerLines())
	for _, line := range lines {
		clearHorizontalRuns(img, line)
		line = adjustLineEndpoints(line)
		line := line.ToPolyline(true).Simplify(1.4)
		path := svgpath.SubPath{
			X: line[0].X,
			Y: line[0].Y,
		}
		for _, point := range line[1:] {
			path.DrawTo = append(path.DrawTo, &svgpath.DrawTo{
				Command: svgpath.LineTo,
				X:       point.X,
				Y:       point.Y,
			})
		}
		horizontalRunPathNode.Path = append(horizontalRunPathNode.Path, &path)
	}

	// Make this second pass more lenient than the first
	pj = NewPointJoiner(10, img.Height, 2)
	//fmt.Println("\n******** Find Vertical Runs")
	FindVerticalRuns(img, pj)
	lines = filterLines(pj.JoinerLines())
	for _, line := range lines {
		line = adjustLineEndpoints(line)
		line := line.ToPolyline(false).Simplify(1.4)
		path := svgpath.SubPath{
			X: line[0].X,
			Y: line[0].Y,
		}
		for _, point := range line[1:] {
			path.DrawTo = append(path.DrawTo, &svgpath.DrawTo{
				Command: svgpath.LineTo,
				X:       point.X,
				Y:       point.Y,
			})
		}
		verticalRunPathNode.Path = append(verticalRunPathNode.Path, &path)
	}

	//cleaner.Simplify(&svg)

	data, err := svg.Marshal()
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func checkReportRun(major, minor, runStart int, runHandler RunHandler) {
	if runStart < 0 {
		return
	}
	runLength := major - runStart
	if runLength <= cfg.VectorizeMaxRunLength {
		runHandler.AddRun(float32(major+runStart)/2, runLength)
	}
}

func FindHorizontalRuns(img *ColorImage, runHandler RunHandler) {
	// To start with, just look for white and black pixels.
	// This will of course need to be expanded to other colors, which could be done
	// trivially by running multiple passes of this alg, one for each color. But it
	// is probably more efficient to look for all colors at the same time.
	i := 0
	for y := 0; y < img.Height; y++ {
		runStart := -1
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
				checkReportRun(x, y, runStart, runHandler)
				runStart = -1
			}
		}
		// check for finished run at end of row
		checkReportRun(img.Width, y, runStart, runHandler)
		runHandler.NextMinor()
	}
}

func FindVerticalRuns(img *ColorImage, runHandler RunHandler) {
	// Note: although it's possible to combine the implementations of this functinon and
	// FindHorizontalRuns, the result would be significantly more complex due to the number
	// of differences. Also this is one of the most performance critical loops in this
	// project, and making the implementation more general would most likely reduce performance.
	for x := 0; x < img.Width; x++ {
		runStart := -1
		for y := 0; y < img.Height; y++ {
			c := img.Data[x+y*img.Width]
			if c == color.Black {
				if runStart == -1 {
					// new run
					runStart = y
				}
			} else {
				// Non-black; check for finished run
				checkReportRun(y, x, runStart, runHandler)
				runStart = -1
			}
		}
		// check for finished run at end of row
		checkReportRun(img.Height, x, runStart, runHandler)
		runHandler.NextMinor()
	}
}
