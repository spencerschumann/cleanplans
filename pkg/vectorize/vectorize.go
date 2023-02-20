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
	"sort"
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

// arcToPath converts a geometry.Arc to an SVG path string representation.
func arcToPath(arc geometry.Arc) string {
	// Arc path format:
	// A rx ry x-axis-rotation large-arc-flag sweep-flag x y

	radius := arc.Start.Distance(arc.Center)

	sweepFlag := 0
	if arc.Clockwise {
		sweepFlag = 1
	}

	largeFlag := 0
	crossProduct := arc.Start.Minus(arc.Center).CrossProductZ(arc.End.Minus(arc.Center))
	if (crossProduct < 1) == arc.Clockwise {
		largeFlag = 1
	}

	return fmt.Sprintf(" M %f %f A %f %f 0 %d %d %f %f ",
		arc.Start.X, arc.Start.Y,
		radius, radius, largeFlag, sweepFlag,
		arc.End.X, arc.End.Y,
	)
}

// NOTE: this can be made generic by declaring it as:
//
//	reverse[T any](input []T)
//
// but gopherjs does not yet support generics, and this
// minor concession is worth making for gopherjs support.
func reverse(input geometry.Polyline) {
	inputLen := len(input)
	inputMid := inputLen / 2

	for i := 0; i < inputMid; i++ {
		j := inputLen - i - 1
		input[i], input[j] = input[j], input[i]
	}
}

func Vectorize(img *ColorImage) string {
	segPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#00aa00;stroke-width:1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	linePathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#aa0000;stroke-width:.5;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	arcPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:none;stroke:#aa0077;stroke-width:.5;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	blobPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:#000000;fill-opacity:0.3;stroke:#ee0000;stroke-width:.1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
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
			&segPathNode,
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
	addSegLine := func(seg geometry.LineSegment) {
		addLineTo(geometry.Polyline{seg.A, seg.B}, &segPathNode)
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

	var hbf, vbf *BlobFinder
	var vBlobs []*Blob
	var unusedRuns []map[*Run]struct{}
	for i := 0; i <= img.Height; i++ {
		unusedRuns = append(unusedRuns, map[*Run]struct{}{})
	}
	// this function obviously and definitely needs to be refactored and broken up. But for right now, it's convenient to have
	// details all together here, for easy addition of debug visualization. But I've added a block here to avoid leaking excessive locals.
	{
		bf := NewBlobFinder(10, img.Width, img.Height)
		bf.TrackRuns = true
		FindHorizontalRuns(img, bf)
		blobs := bf.Blobs()

		tBlobs, connections, tRuns := Transpose(blobs, img.Width, img.Height)
		_ = tBlobs
		_ = connections

		// Colorize based on shortest run direction
		for _, blob := range blobs {
			for _, run := range blob.Runs {
				width := run.X2 - run.X1
				for x := int(run.X1); x < int(run.X2); x++ {
					tRow := tRuns[x]
					ti := sort.Search(len(tRow), func(i int) bool {
						return run.Y <= tRow[i].X2
					})
					tRun := tRow[ti]
					tWidth := tRun.X2 - tRun.X1

					if tWidth <= width {
						run.Eclipsed = true
						y := int(run.Y)
						unusedRuns[y][run] = struct{}{}
					}
					if width <= tWidth {
						tRun.Eclipsed = true
					}

					ii := 0
					if blob.Transposed {
						ii = int(run.Y) + x*img.Width
					} else {
						ii = x + int(run.Y)*img.Width
					}
					if tWidth < width {
						img.Data[ii] = color.LightPurple
					} else if width < tWidth {
						img.Data[ii] = color.LightGray
					}
				}
			}
		}

		hbf = NewBlobFinder(10, img.Width, img.Height)
		//hbf.TrackRuns = true // TODO: do I ever end up setting this to false?
		for _, row := range bf.Runs {
			for _, run := range row {
				if !run.Eclipsed {
					hbf.AddRun(run.X1, run.X2) // TODO: pass a Run struct to AddRun instead?
				}
			}
			hbf.NextY()
		}
		vbf = NewBlobFinder(10, img.Height, img.Width)
		for _, row := range tRuns {
			for _, tRun := range row {
				if !tRun.Eclipsed {
					vbf.AddRun(tRun.X1, tRun.X2)
				}
			}
			vbf.NextY()
		}
		vBlobs = vbf.Blobs()
		for _, blob := range vbf.Blobs() {
			blob.Transposed = true
		}

		// Reset all eclipsed flags
		for _, row := range append(hbf.Runs, vbf.Runs...) {
			for _, run := range row {
				run.Eclipsed = false
			}
		}
	}

	for _, blob := range append(hbf.Blobs(), vBlobs...) {
		// Note: won't even need this, since all pixels will be accounted for as blobs.
		// But it's still useful for testing, to gray out the detected blobs.
		//clearHorizontalRuns(img, blob)

		outline := blob.Outline(0.2)
		var arc geometry.Arc
		//var circle geometry.Circle

		var polyline geometry.Polyline
		var segs []geometry.LineSegment
		count := float64(len(blob.Runs))
		wAvg := blob.Runs.AverageWidth()
		used := false
		if wAvg*1.1 < count {
			polyline, segs = blob.ToPolyline()
			//arc = blob.BestFitArc()
			//circle = blob.BestFitCircle()

			if len(polyline) > 0 {
				// a line was generated; mark these runs as used
				for _, run := range blob.Runs {
					run.Eclipsed = true
				}
				used = true
			}
		}

		if used {
			if blob.Transposed {
				for i := range outline {
					p := &outline[i]
					p.X, p.Y = p.Y, p.X
				}
				addTransposedBlobOutline(outline)
			} else {
				addBlobOutline(outline)
			}
		} else if !blob.Transposed {
			// Make note of additional unused horizontal runs
			for _, run := range blob.Runs {
				unusedRuns[int(run.Y)][run] = struct{}{}
			}
		}

		if (arc != geometry.Arc{}) {
			arcPathNode.D += arcToPath(arc)
			//addCircle(circle)
		}
		addLine(polyline)

		for _, seg := range segs {
			addSegLine(seg)
		}
	}

	// Take the unused horizontal runs as a baseline for the remainder.
	// Then take the eclipsed vertical runs and transpose them, to find
	// the set of corresponding horizontal runs that were eclipsed from the vertical
	// set. Note that these runs may overlap actual horizontal runs only partially.
	// Then, subtract the transposed vertical eclipsed runs from the unused
	// horizontal runs.
	{
		// Sort unused horizontal runs
		hRuns := make([][]*Run, img.Height+1)
		for y, row := range unusedRuns {
			for run := range row {
				hRuns[y] = append(hRuns[y], run)
			}
			sort.Slice(hRuns[y], func(i, j int) bool {
				return hRuns[y][i].X1 < hRuns[y][j].X1
			})
			/*for _, run := range hRuns[y] {
				fmt.Println("Unused run:", run)
			}*/
		}

		// Get eclipsed vertical blobs
		var evBlobs []*Blob
		for _, blob := range vBlobs {
			if blob.Runs[0].Eclipsed {
				evBlobs = append(evBlobs, blob)
			}
		}

		// TODO: not needing tBlobs here, computation for them is wasted. Need to refactor Transpose() for this use case.
		_, _, hvRuns := Transpose(evBlobs, img.Height, img.Width)

		/*for y := range hvRuns {
			for _, run := range hvRuns[y] {
				fmt.Println("Transposed used vertical run:", run)
			}
		}*/

		// Subtract hvRuns out of hRuns
		// First, a length check - should match
		if len(hvRuns) != len(hRuns) {
			fmt.Printf("len(hRuns)=%d, len(hvRuns)=%d\n", len(hRuns), len(hvRuns))
			panic("hvRuns length != hRuns length")
		}
		bf := NewBlobFinder(10, img.Width, img.Height)
		for i := range hRuns {
			uRow := hRuns[i]
			hvRow := hvRuns[i]
			hvi := 0
			for _, run := range uRow {
				// scan forward to the next hvi that could overlap this run
				for run.X1 < run.X2 {
					for hvi < len(hvRow) && hvRow[hvi].X2 <= run.X1 {
						hvi++
					}
					if hvi < len(hvRow) {
						hvRun := hvRow[hvi]
						//fmt.Println("run=", run, "hvrun=", hvRun)
						if run.X2 <= hvRun.X1 {
							// no overlap
							//fmt.Println("  no overlap, add full run", run)
							bf.AddRun(run.X1, run.X2)
							break
						} else {
							// add first chunk of run, if it exists
							if run.X1 < hvRun.X1 {
								//fmt.Println("  overlap, add", run.X1, "to", hvRun.X1)
								bf.AddRun(run.X1, hvRun.X1)
								// mark this chunk used
								run.X1 = hvRun.X1
							}
							// remove overlapping portion
							if hvRun.X2 <= run.X2 {
								//fmt.Println("  remove chunk up to", hvRun.X2, "from run")
								run.X1 = hvRun.X2
							}
						}
					} else {
						// no more hv runs on this row
						//fmt.Println("  Add run", run)
						bf.AddRun(run.X1, run.X2)
						break
					}
				}
			}
			bf.NextY()
		}
		for _, blob := range bf.Blobs() {
			addBlobOutline(blob.Outline(0.3))

			var polyline geometry.Polyline
			var segs []geometry.LineSegment
			count := float64(len(blob.Runs))
			wAvg := blob.Runs.AverageWidth()
			if wAvg*1.1 < count {
				polyline, segs = blob.ToPolyline()
				addLine(polyline)
				for _, seg := range segs {
					addSegLine(seg)
				}
			}
		}
	}

	svg.Children = append(svg.Children, &arcPathNode)

	/*for _, c := range connections {
		if c.A.Transposed {
			addPoint(c.Location.Y, c.Location.X)
		} else {
			addPoint(c.Location.X, c.Location.Y)
		}
	}*/

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
