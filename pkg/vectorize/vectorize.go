package vectorize

import (
	"cleanplans/pkg/cleaner"
	"cleanplans/pkg/color"
	"cleanplans/pkg/geometry"
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"fmt"
	"image"
	imgcolor "image/color"
	"strconv"
)

// Terrible name...if this works, I need to change names to avoid collisions with the standard Go image and color packages.
type ColorImage struct {
	Width  int
	Height int
	Data   []color.Color
}

/*type Point struct {
	X float32
	Y float32
}

type Line []Point*/

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
	AddRun(x1, x2 float64)
	NextY()
}

// This helps close gaps, but it also doesn't take into consideration line slope. It may
// be better to do away with this when adding the code to join line segments to eliminate
// gaps.
/*func adjustLineEndpoints(line Blob) Blob {
	last := len(line) - 1
	for i := 1; i < last; i++ {
		line[i].Y += 0.5
	}
	line[last].Y += 1
	return line
}*/

func clearHorizontalRuns(img *ColorImage, blob *Blob) {
	if blob.Transposed {
		for _, run := range blob.Runs {
			x := int(run.Y)
			for y := int(run.X1); y < int(run.X2); y++ {
				img.Data[x+y*img.Width] = color.LightGray
			}
		}
	} else {
		for _, run := range blob.Runs {
			for x := int(run.X1); x < int(run.X2); x++ {
				img.Data[x+int(run.Y)*img.Width] = color.LightGray
			}
		}
	}
}

/*func filterLines(pj *BlobFinder, lines []Blob) []Blob {
	var output []Blob
	for _, line := range lines {
		output = append(output, filterLine(pj, line)...)
	}
	return output
}

func filterLine(pj *BlobFinder, line Blob) []Blob {
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

	// Only allow widths of median +/- 2.5 (one pixel on each side, plus a little slop) or 20%
	widthOk := func(width int) bool {
		diff := math.Abs(float64(median - width))
		return diff <= 2.5 || diff < (float64(median)*.2)
	}

	bestRunStart := -1
	var lines []Blob
	checkReportRun := func(i int) {
		if bestRunStart < 0 {
			return
		}
		// TODO: may want to further trim the beginning and end of the subline with more strict requirements
		subLine := line[bestRunStart:i]
		lines = append(lines, subLine)
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

func trimLines(lines []Blob) []Blob {
	var result []Blob
	for _, line := range lines {
		totalWidth := 0.0
		for _, pt := range line {
			totalWidth += float64(pt.Width)
		}
		avgWidth := totalWidth / float64(len(line))

		// Trim avgWidth segments off the beginning and end of the line; if there's nothing left, remove it completely.
		trim := int(avgWidth + 0.5)
		if trim == 0 {
			trim = 1
		}
		if len(line) > trim*3 {
			result = append(result, line[trim:len(line)-trim])
		}
	}
	return result
}*/

func reverse[T any](input []T) {
	inputLen := len(input)
	inputMid := inputLen / 2

	for i := 0; i < inputMid; i++ {
		j := inputLen - i - 1
		input[i], input[j] = input[j], input[i]
	}
}

func Vectorize(img *ColorImage) string {
	linePathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#aa0000;stroke-width:.5;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	blobPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:#000000;fill-opacity:0.5;stroke:#ee0000;stroke-width:.1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	transposedBlobPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:#000033;fill-opacity:0.2;stroke:#00aaaa;stroke-width:.1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	rectPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#00ee00;stroke-width:.2;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	svg := cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "svg"},
		Children: []*cleaner.SVGXMLNode{
			&linePathNode,
			&blobPathNode,
			&transposedBlobPathNode,
			&rectPathNode,
		},

		// Note: not using a unit specifier here for display, to match up with the png image. For
		// the final output SVG (if that is the format I go with), these will need to be mapped to mm.
		Width:  strconv.Itoa(img.Width),
		Height: strconv.Itoa(img.Height),
	}

	addPoint := func(x, y float64) {
		svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
			XMLName: xml.Name{Local: "circle"},
			CX:      x,
			CY:      y,
			Radius:  0.5,
			Styles:  "fill:#00ff00",
		})
	}

	//lineSet := NewLineSet(float64(img.Width), float64(img.Height))

	addCircle := func(circle geometry.Circle) {
		svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
			XMLName: xml.Name{Local: "circle"},
			CX:      circle.Center.X,
			CY:      circle.Center.Y,
			Radius:  circle.Radius,
			Styles:  "fill:none;stroke:#00aa00;stroke-width:.5;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		})
	}

	addLineTo := func(line geometry.Polyline, node *cleaner.SVGXMLNode) {
		if len(line) < 2 {
			return
		}
		path := svgpath.SubPath{
			X: line[0].X,
			Y: line[0].Y,
		}
		//addPoint(line[0].X, line[0].Y)
		for _, point := range line[1:] {
			path.DrawTo = append(path.DrawTo, &svgpath.DrawTo{
				Command: svgpath.LineTo,
				X:       point.X,
				Y:       point.Y,
			})
			//addPoint(point.X, point.Y)
		}
		node.Path = append(node.Path, &path)
	}
	addBlobOutline := func(line geometry.Polyline) {
		addLineTo(line, &blobPathNode)
	}
	addTransposedBlobOutline := func(line geometry.Polyline) {
		addLineTo(line, &transposedBlobPathNode)
	}
	addLine := func(line geometry.Polyline) {
		addLineTo(line, &linePathNode)
	}
	addRectLine := func(line geometry.Polyline) {
		addLineTo(line, &rectPathNode)
	}

	// Ignore unused warnings
	_ = addCircle
	_ = addLine
	_ = addTransposedBlobOutline
	_ = addPoint
	_ = addRectLine

	// First pass: find lines up to 45 degrees off vertical
	bf := NewBlobFinder(10, img.Width, img.Height)
	FindHorizontalRuns(img, bf)
	blobs := bf.Blobs()
	//lines = filterLines(bf, lines)

	// Double transpose, to do the transposition but then to display in normal coords
	blobs = Transpose(blobs, img.Width, img.Height)

	for _, blob := range blobs {
		// Note: won't even need this, since all pixels will be accounted for as blobs.
		// But it's still useful for testing, to gray out the detected blobs.
		clearHorizontalRuns(img, blob)

		margin := 0.2
		x1, x2 := blob.Runs[0].X1, blob.Runs[0].X2
		left := geometry.Polyline{{X: x1 + margin, Y: blob.Runs[0].Y + margin}}
		right := geometry.Polyline{{X: x2 - margin, Y: blob.Runs[0].Y + margin}}
		lastRun := blob.Runs[0]
		for _, run := range blob.Runs[1:] {
			if run.X1 < x1 {
				left = append(left, geometry.Point{X: x1 + margin, Y: run.Y + margin})
				left = append(left, geometry.Point{X: run.X1 + margin, Y: run.Y + margin})
			} else if x1 < run.X1 {
				left = append(left, geometry.Point{X: x1 + margin, Y: lastRun.Y + 1 - margin})
				left = append(left, geometry.Point{X: run.X1 + margin, Y: lastRun.Y + 1 - margin})
			}
			if run.X2 < x2 {
				right = append(right, geometry.Point{X: x2 - margin, Y: lastRun.Y + 1 - margin})
				right = append(right, geometry.Point{X: run.X2 - margin, Y: lastRun.Y + 1 - margin})
			} else if x2 < run.X2 {
				right = append(right, geometry.Point{X: x2 - margin, Y: run.Y + margin})
				right = append(right, geometry.Point{X: run.X2 - margin, Y: run.Y + margin})
			}
			x1 = run.X1
			x2 = run.X2
			lastRun = run
		}
		left = append(left, geometry.Point{X: lastRun.X1 + margin, Y: lastRun.Y + 1 - margin})
		right = append(right, geometry.Point{X: lastRun.X2 - margin, Y: lastRun.Y + 1 - margin})
		line := geometry.Polyline(left)
		reverse(right)
		line = append(line, right...)
		// Close the loop
		line = append(line, line[0])

		if blob.Transposed {
			for i := range line {
				p := &(line[i])
				p.X, p.Y = p.Y, p.X
			}
			addTransposedBlobOutline(line)
		} else {
			addBlobOutline(line)
		}
	}

	for _, c := range bf.Connections() {
		addPoint(c.Location.X, c.Location.Y)
	}

	data, err := svg.Marshal()
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func checkReportRun(x, y, runStart int, runHandler RunHandler) {
	if runStart < 0 {
		return
	}
	//runLength := x - runStart
	//if runLength <= cfg.VectorizeMaxRunLength {
	runHandler.AddRun(float64(runStart), float64(x))
	//}
}

func colorOk(c color.Color) bool {
	switch c {
	case color.Black:
		return true
	case color.Blue, color.Gray, color.Green, color.Red:
		return false
	default:
		return false
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
			if colorOk(c) {
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
		runHandler.NextY()
	}
}
