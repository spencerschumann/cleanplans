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
	"math"
	"sort"
	"strconv"
	"time"
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
func PDFJSImageToColorImage(image []uint8, width, height, bitsPerPixel int) *ColorImage {
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
			// Assume most of the image is white colored background; optimize for long runs of white.
			// On the WASM build, this cuts down PDFJSImageToColorImage's run time by around half.
			if image[i] == 0xff {
				k := i + 1
				for k < size && image[k] == 0xff {
					k++
				}
				run := (k - i) / 3
				for k := i; k < j+run; k++ {
					data[k] = color.White
				}
				j += run
				i += run * 3
				if i == size {
					break
				}
			}
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
	AddRun(run *Run)
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

func timeDeltaMS(t1, t2 time.Time) float64 {
	return float64(t2.Sub(t1)) / float64(time.Millisecond)
}

var _checkpoint_lastTime time.Time

func Checkpoint(msg string) {
	now := time.Now()
	fmt.Printf("Time for %s: %gms\n", msg, timeDeltaMS(_checkpoint_lastTime, now))
	_checkpoint_lastTime = now
}

func Vectorize(img *ColorImage) string {
	_checkpoint_lastTime = time.Now()

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
	verticalMarginPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:#dddddd;fill-opacity:0.8;stroke:#ee0000;stroke-width:.1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	horizontalMarginPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:#000000;fill-opacity:0.3;stroke:#ee0000;stroke-width:.1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
		Category: cleaner.CategoryFullCut,
	}
	marginIntersectionPathNode := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "path"},
		Styles:   "fill:#ff0000;fill-opacity:0.3;stroke:#00ee00;stroke-width:1;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:4;stroke-opacity:1",
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
			&verticalMarginPathNode,
			&horizontalMarginPathNode,
			&marginIntersectionPathNode,
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
	addBlobOutline := func(line geometry.Polyline, node *cleaner.SVGXMLNode) {
		addLineTo(line, node)
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
	_ = addPoint
	_ = addRectLine

	Checkpoint("Vectorize preamble")
	allRuns := FindAllHorizontalRuns(img)
	for c := color.White; c <= color.LightPurple; c++ {
		fmt.Printf("Row count for color %d: %d\n", c, len(allRuns[c]))
	}
	Checkpoint("FindAllHorizontalRuns")

	if true {
		// Try to find a clipping rectangle to ignore boilerplate - assume the
		// content will be separated from the boilerplate by a rectangular margin.

		minRatio := 0.9

		yFactor := 1
		// Note: this factor isn't quite working, but I think it could give a good speedup.
		// The problem is that after transposing, some of the lines get skipped.
		if img.Height > 1000 {
			yFactor = 10
		}

		// Start by finding horizontal near-rectangles
		vMarginBF := NewBlobFinder(img.Width, img.Width, img.Height/yFactor)
		minWidth := float64(img.Width) * minRatio
		for y := 0; y < img.Height; y += yFactor {
			row := allRuns[color.White][y]
			for _, run := range row {
				width := run.X2 - run.X1
				if width > minWidth {
					run := *run
					run.Y = float64(y / yFactor)
					vMarginBF.AddRun(&run)
				}
			}
			vMarginBF.NextY()
		}
		Checkpoint("vMarginBF blob finder")

		/*for _, blob := range vMarginBF.Blobs() {
			outline := blob.Outline(0.1, false)
			for i := range outline {
				outline[i].Y *= float64(yFactor)
			}
			addBlobOutline(outline, &verticalMarginPathNode)
		}*/

		// Gather together "big enough" runs of non-background in the middle; call these the content region.
		unblob := func(blobs []*Blob, maxY float64) Runs {
			sort.Slice(blobs, func(i, j int) bool {
				return blobs[i].Runs[0].Y < blobs[j].Runs[0].Y
			})
			var runs Runs
			yStart := 0.0
			for _, blob := range blobs {
				firstY, lastY := blob.Runs[0].Y, blob.Runs[len(blob.Runs)-1].Y+1
				//fmt.Println("firstY, lastY, yStart:", firstY, lastY, yStart)
				if yStart < firstY {
					runs = append(runs, &Run{X1: yStart, X2: firstY})
				}
				yStart = lastY
			}
			if yStart < maxY {
				runs = append(runs, &Run{X1: yStart, X2: maxY})
			}
			return runs
		}

		coalesce := func(runs Runs, width float64) Run {
			firstX := math.Inf(+1)
			lastX := math.Inf(-1)

			// Keep anything that's "big" or near the center
			for _, run := range runs {
				rw := run.X2 - run.X1
				rc := (run.X2 + run.X1) / 2
				if rw > width*0.1 || (width*0.2 < rc && rc < width*0.8) {
					firstX = math.Min(firstX, run.X1)
					lastX = math.Max(lastX, run.X2)
				}
			}

			return Run{X1: firstX, X2: lastX}
		}

		// first scan and unblob and find the vertical content range; then transform those into blobs and transpose.
		// This will avoid unnecessary blob processing within high-cost text areas at the top and bottom of the page.

		vMarginRuns := unblob(vMarginBF.Blobs(), float64(img.Height/yFactor))
		vContentRun := coalesce(vMarginRuns, float64(img.Height/yFactor))
		// calculate minWidth for the transposed blob finder before transforming the coords
		minWidth = vContentRun.X2 - vContentRun.X1
		vContentRun.X1 = math.Max(0, (vContentRun.X1-1)*float64(yFactor)+1)
		vContentRun.X2 *= float64(yFactor)
		if false {
			line := geometry.Polyline{
				{X: 0, Y: vContentRun.X1},
				{X: 0, Y: vContentRun.X2},
				{X: float64(img.Width), Y: vContentRun.X2},
				{X: float64(img.Width), Y: vContentRun.X1},
				{X: 0, Y: vContentRun.X1},
			}
			addLineTo(line, &verticalMarginPathNode)
		}
		Checkpoint("finalize vContentRun")

		// First attempt - transpose and blob find. Commented out for now for reference.
		/*bf := NewBlobFinder(200, img.Width, img.Height/yFactor)
		for y := 0; y < img.Height; y += yFactor {
			row := allRuns[color.White][y]
			if len(row) > 0 && vContentRun.X1 <= row[0].Y && row[0].Y < vContentRun.X2 {
				for _, run := range row {
					run := *run
					// TODO: can also compress horizontally here - some runs may disappear when doing this
					run.Y = float64(y / yFactor)
					bf.AddRun(&run)
				}
			}
			bf.NextY()
		}
		Checkpoint("gather runs to transpose")

		blobs := bf.Blobs()
		if false {
			for _, blob := range blobs {
				fmt.Println("Blob")
				for _, run := range blob.Runs {
					fmt.Println("  Run:", *run)
				}
			}
		}
		Checkpoint("bf.Blobs()")
		tRuns := Transpose(blobs, img.Width, img.Height/yFactor)
		Checkpoint("Transpose")
		hMarginBF := NewBlobFinder(50, img.Height/yFactor, img.Width)
		// TODO: this is a new common pattern - need to move it to a method of BlobFinder
		for _, row := range tRuns {
			for _, run := range row {
				width := run.X2 - run.X1
				//fmt.Println("Transposed run:", *run, "width:", width, "minWidth:", minWidth, "totalWidth:", img.Height/yFactor)
				if width >= minWidth {
					//fmt.Println("Adding transposed run", *run)
					hMarginBF.AddRun(run)
				}
			}
			hMarginBF.NextY()
		}

		hMarginRuns := unblob(hMarginBF.Blobs(), float64(img.Width))
		if true {
			for _, blob := range hMarginBF.Blobs() {
				outline := blob.Outline(0.1, false)
				for i := range outline {
					p := &outline[i]
					p.X, p.Y = p.Y, p.X*float64(yFactor)
				}
				addBlobOutline(outline, &horizontalMarginPathNode)
			}
		}
		hMarginRun := coalesce(hMarginRuns, float64(img.Width))*/

		hMarginRun := Run{X2: float64(img.Width)}

		// New attempt on the horizontal margins: find vertical swatches directly from the runs
		{
			indices := make([]int, img.Height)
			//x := 0.0
			yStart := int(vContentRun.X1)
			yEnd := int(vContentRun.X2)

			scanX := 0.0
			for scanX < float64(img.Width) {
				// find the nearest start of run from the current x pos

				// find the next x position where each row has started a run, and find the largest of all these.
				largestX := scanX
				for y := yStart; y < yEnd; y++ {
					row := allRuns[color.White][y]
					for indices[y] < len(row) {
						run := row[indices[y]]
						if scanX < run.X1 {
							// next run is past scanX
							largestX = run.X1
							break
						}
						if run.X1 <= scanX && scanX < run.X2 {
							// found a run that holds scanX
							largestX = math.Max(largestX, row[indices[y]].X1)
							break
						}
						indices[y]++ // TODO: is it worth trying to do better than a linear scan here?
					}
					if indices[y] >= len(row) {
						// No runs left on this row
						largestX = float64(img.Width)
					}
				}
				fmt.Println("...scanX=", scanX, "largestX=", largestX)
				if scanX != largestX {
					// Keep trying until all rows have a run at the same point
					scanX = largestX
					continue
				}
				if scanX >= float64(img.Width) {
					// Not sure this is needed, actually
					break
				}
				fmt.Println("Next X start:", largestX)

				// Now scan forward to the earliest run end of any row
				nextX := float64(img.Width)
				for y := yStart; y < yEnd; y++ {
					run := allRuns[color.White][y][indices[y]]
					nextX = math.Min(nextX, run.X2)
				}
				fmt.Println("    X end:", nextX)
				//x = nextX
				scanX = nextX // TODO: maybe x and scanX are the same?
			}
		}

		fmt.Println("coalesced hMarginRun:", hMarginRun)
		if false {
			line := geometry.Polyline{
				{X: hMarginRun.X1, Y: 0},
				{X: hMarginRun.X2, Y: 0},
				{X: hMarginRun.X2, Y: float64(img.Height)},
				{X: hMarginRun.X1, Y: float64(img.Height)},
				{X: hMarginRun.X1, Y: 0},
			}
			addLineTo(line, &horizontalMarginPathNode)
		}
		Checkpoint("process hMarginRuns")

		addLineTo(geometry.Polyline{
			{X: 0, Y: 0},
			{X: 0, Y: float64(img.Height)},
			{X: float64(img.Width), Y: float64(img.Height)},
			{X: float64(img.Width), Y: 0},
			{X: 0, Y: 0},
		}, &verticalMarginPathNode)
		addLineTo(geometry.Polyline{
			{X: hMarginRun.X1, Y: vContentRun.X1},
			{X: hMarginRun.X2, Y: vContentRun.X1},
			{X: hMarginRun.X2, Y: vContentRun.X2},
			{X: hMarginRun.X1, Y: vContentRun.X2},
			{X: hMarginRun.X1, Y: vContentRun.X1},
		}, &verticalMarginPathNode)

		// Crop all other runs
		for c, rows := range allRuns {
			filteredRows := make([]Runs, len(rows))
			for y, row := range rows {
				if vContentRun.X1 <= float64(y) && float64(y) < vContentRun.X2 {
					for _, run := range row {
						if hMarginRun.X1 < run.X2 && run.X1 < hMarginRun.X2 {
							run.X1 = math.Max(run.X1, hMarginRun.X1)
							run.X2 = math.Min(run.X2, hMarginRun.X2)
							filteredRows[y] = append(filteredRows[y], run)
						}
					}
				}
			}
			allRuns[c] = filteredRows
		}
		Checkpoint("crop all other runs")
	}

	// TODO: clean up runs - remove single points and single point voids

	var hbf, vbf *BlobFinder
	var vBlobs []*Blob
	unusedRuns := make([]map[*Run]struct{}, img.Height)
	for i := 0; i < img.Height; i++ {
		unusedRuns[i] = map[*Run]struct{}{}
	}
	// this function obviously and definitely needs to be refactored and broken up. But for right now, it's convenient to have
	// details all together here, for easy addition of debug visualization. But I've added a block here to avoid leaking excessive locals.
	{
		bf := NewBlobFinder(100, img.Width, img.Height)
		//bf.TrackRuns = true
		//FindHorizontalRuns(img, bf)
		for _, row := range allRuns[color.Black] {
			for _, run := range row {
				bf.AddRun(run)
			}
			bf.NextY()
		}
		blobs := bf.Blobs()

		tRuns := Transpose(blobs, img.Width, img.Height)

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

		hbf = NewBlobFinder(100, img.Width, img.Height)
		for _, row := range allRuns[color.Black] {
			for _, run := range row {
				if !run.Eclipsed {
					hbf.AddRun(run)
				}
			}
			hbf.NextY()
		}
		vbf = NewBlobFinder(100, img.Height, img.Width)
		for _, row := range tRuns {
			for _, tRun := range row {
				if !tRun.Eclipsed {
					vbf.AddRun(tRun)
				}
			}
			vbf.NextY()
		}
		vBlobs = vbf.Blobs()
		for _, blob := range vbf.Blobs() {
			blob.Transposed = true
		}

		// Reset all eclipsed flags
		for _, row := range append(allRuns[color.Black], tRuns...) {
			for _, run := range row {
				run.Eclipsed = false
			}
		}
	}

	for _, blob := range append(hbf.Blobs(), vBlobs...) {
		// Note: won't even need this, since all pixels will be accounted for as blobs.
		// But it's still useful for testing, to gray out the detected blobs.
		//clearHorizontalRuns(img, blob)

		outline := blob.Outline(0.2, blob.Transposed)
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
				addBlobOutline(outline, &transposedBlobPathNode)
			} else {
				addBlobOutline(outline, &blobPathNode)
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
		hRuns := make([][]*Run, img.Height)
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
		hvRuns := Transpose(evBlobs, img.Height, img.Width)

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
				// don't corrupt earlier runs, in case they're needed later
				run := *run
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
							// add a snapshot of the current state of this run copy
							run := run
							bf.AddRun(&run)
							break
						} else {
							// add first chunk of run, if it exists
							if run.X1 < hvRun.X1 {
								//fmt.Println("  overlap, add", run.X1, "to", hvRun.X1)
								bf.AddRun(&Run{X1: run.X1, X2: hvRun.X1, Y: run.Y})
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
						// add a snapshot of the current state of this run copy
						run := run
						bf.AddRun(&run)
						break
					}
				}
			}
			bf.NextY()
		}
		// The runs in hRuns have been modified and some have been added, so hRuns is no longer an accurate list.
		hRuns = nil

		for _, blob := range bf.Blobs() {
			addBlobOutline(blob.Outline(0.3, false), &blobPathNode)

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

// FindAllHorizontalRuns returns a slice of runs for every color.
func FindAllHorizontalRuns(img *ColorImage) [][]Runs {
	// Prepare for up to 256 colors
	allRuns := make([][]Runs, 256)

	checkReportRun := func(x, y, runStart int, c color.Color) {
		if runStart < 0 {
			return
		}
		i := int(c)

		if len(allRuns[i]) == 0 {
			allRuns[i] = make([]Runs, img.Height)
		}

		allRuns[i][y] = append(allRuns[i][y], &Run{X1: float64(runStart), X2: float64(x), Y: float64(y)})
	}

	imgIndex := 0
	for y := 0; y < img.Height; y++ {
		runStarts := make([]int, 256)
		for i := range runStarts {
			runStarts[i] = -1
		}
		lastColor := -1
		for x := 0; x < img.Width; x++ {
			// TODO: in the final implementation, pixels can be fed in directly as they're
			// decoded rather than storing them in a separate image slice.
			c := int(img.Data[imgIndex])
			imgIndex++
			if lastColor != c {
				// new color; check for finished run
				if lastColor >= 0 {
					checkReportRun(x, y, runStarts[lastColor], color.Color(lastColor))
					runStarts[lastColor] = -1
				}
				// Start new run
				runStarts[c] = x
			}
			lastColor = c
		}
		// check for finished run at end of row
		if lastColor >= 0 {
			checkReportRun(img.Width, y, runStarts[lastColor], color.Color(lastColor))
		}
	}
	return allRuns
}
