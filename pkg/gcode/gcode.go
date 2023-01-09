package gcode

import (
	"cleanplans/pkg/cleaner"
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"fmt"
	"log"
	"math"
)

type point struct {
	x float64
	y float64
}

type curve struct {
	p0 point
	p1 point
	p2 point
	p3 point
}

// splitCurve splits the curve at the mid point (t=0.5)
func splitCurve(c curve) (curve, curve) {
	mid := func(p1, p2 point) point {
		return point{
			x: (p1.x + p2.x) / 2,
			y: (p1.y + p2.y) / 2,
		}
	}

	p4 := mid(c.p0, c.p1)
	p5 := mid(c.p1, c.p2)
	p6 := mid(c.p2, c.p3)
	p7 := mid(p4, p5)
	p8 := mid(p5, p6)
	p9 := mid(p7, p8)

	return curve{p0: c.p0, p1: p4, p2: p7, p3: p9},
		curve{p0: p9, p1: p8, p2: p6, p3: c.p3}
}

// isFlat returns true if the curve is flat enough to be replaced with a line segment.
func isFlat(c curve) bool {
	// TODO: make this configurable
	tol := 0.1 // 0.1 millimeter max error

	ax := 3*c.p1.x - 2*c.p0.x - c.p3.x
	ay := 3*c.p1.y - 2*c.p0.y - c.p3.y
	bx := 3*c.p2.x - c.p0.x - 2*c.p3.x
	by := 3*c.p2.y - c.p0.y - 2*c.p3.y

	maxX := math.Max(ax*ax, bx*bx)
	maxY := math.Max(ay*ay, by*by)

	return (maxX+maxY)/16 <= tol*tol
}

// subdivide subdivides the curve into a series of line segments, described by a list of points
func subdivide(c curve) []point {
	if isFlat(c) {
		return []point{c.p0, c.p3}
	}
	a, b := splitCurve(c)
	aSegs := subdivide(a)
	bSegs := subdivide(b)
	// aSegs ends with the same point that bSegs begins with, so remove one to avoid duplicates
	return append(aSegs, bSegs[1:]...)
}

func curvesToLines(svg *cleaner.SVGXMLNode) {
	var iterate func(svg *cleaner.SVGXMLNode)
	iterate = func(svg *cleaner.SVGXMLNode) {
		for _, child := range svg.Children {
			for _, path := range child.Path {
				hasCurves := false
				for _, drawTo := range path.DrawTo {
					if drawTo.Command == svgpath.CurveTo {
						hasCurves = true
						break
					}
				}

				if hasCurves {
					var segs []*svgpath.DrawTo
					x, y := path.StartPoint()
					for _, drawTo := range path.DrawTo {
						if drawTo.Command == svgpath.LineTo {
							segs = append(segs, drawTo)
						} else if drawTo.Command == svgpath.CurveTo {
							curve := curve{
								p0: point{x: x, y: y},
								p1: point{x: drawTo.X1, y: drawTo.Y1},
								p2: point{x: drawTo.X2, y: drawTo.Y2},
								p3: point{x: drawTo.X, y: drawTo.Y},
							}
							// The first point will be (x,y), which is redundant; remove it.
							for _, point := range subdivide(curve)[1:] {
								segs = append(segs, &svgpath.DrawTo{
									Command: svgpath.LineTo,
									X:       point.x,
									Y:       point.y,
								})
							}
						} else {
							// ERROR!
							// TODO: handle other types of drawing operations
						}
						x, y = drawTo.X, drawTo.Y
					}
					path.DrawTo = segs
				}
			}
			iterate(child)
		}
	}
	iterate(svg)
}

type machineState struct {
	X float64
	Y float64
	Z float64
}

func GenerateSVGWithPaths(svg *cleaner.SVGXMLNode) {
	dashedStyle := "fill:none;stroke:#c900ce;stroke-width:0.5;stroke-linecap:butt;stroke-linejoin:miter;stroke-opacity:1;stroke-miterlimit:4;stroke-dasharray:0.5,0.5;stroke-dashoffset:0"

	// Place dashed lines between each segment to show the pen-up "travel" movements
	travel := &cleaner.SVGXMLNode{
		XMLName: xml.Name{
			Space: "http://www.w3.org/2000/svg",
			Local: "path",
		},
		ID:     "travel_indicators",
		Styles: dashedStyle,
	}

	totalTravel := 0.0
	lastX, lastY := 0.0, svg.HeightInMM()
	travelTo := func(x, y float64) {
		dx := lastX - x
		dy := lastY - y
		totalTravel += math.Sqrt(dx*dx + dy*dy)
		travel.Path = append(travel.Path, &svgpath.SubPath{
			X: lastX,
			Y: lastY,
			DrawTo: []*svgpath.DrawTo{
				{
					Command: svgpath.LineTo,
					X:       x,
					Y:       y,
				},
			},
		})
	}

	var iterate func(svg *cleaner.SVGXMLNode)
	iterate = func(svg *cleaner.SVGXMLNode) {
		//fmt.Fprintf(os.Stderr, "Entering node with %d children\n", len(svg.Children))
		for _, child := range svg.Children {
			for _, path := range child.Path {
				dist := math.Abs(path.X-lastX) + math.Abs(path.Y-lastY)
				if dist > 0.1 {
					travelTo(path.X, path.Y)
				}
				lastX, lastY = path.EndPoint()
			}
			iterate(child)
		}
	}
	iterate(svg)
	travelTo(0, svg.HeightInMM())
	svg.Children = append(svg.Children, travel)

	//fmt.Fprintln(os.Stderr, "Total travel:", totalTravel)

	outXML, err := svg.Marshal()
	if err != nil {
		log.Fatalf("marshal error: %s", err)
	}
	fmt.Println(string(outXML))
}

func Generate(svg *cleaner.SVGXMLNode) {
	cleaner.SortPaths(svg)
	// TODO: configurable - if the gcode processor supports curves, these can be left
	// as curves. If it supports arcs but not curves, it would be better to convert them
	// to arcs instead of lines.
	curvesToLines(svg)

	if false {
		GenerateSVGWithPaths(svg)
		return
	}

	state := &machineState{
		X: 0,
		Y: 0,
		Z: 0,
	}

	xHome := 0.0
	yHome := 0.0
	xyTravelRate := 15000.0
	xyFeedRate := 2000.0
	zFeedRate := 5000.0
	spindleSpeed := 7500.0
	penUpZ := 2.0
	penFullRetractZ := 5.0

	// Output gcode header
	fmt.Println("G21 (metric ftw)")
	fmt.Println("G90 (absolute mode)")

	// For classic TimSav
	//fmt.Printf("G92 X%.2f Y%.2f Z%.2f (you are here)\n", xHome, yHome, zHeight)

	// For TimSav+ with homing switches
	fmt.Println("$H (home all axes)")

	fmt.Printf("G0 F%0.2f (Travel Feed Rate)\n", xyTravelRate)

	fmt.Printf("G1 Z%.2f F%.2f (Retract pen)\n", penFullRetractZ, zFeedRate)
	fmt.Printf("M3 S%0.0f (Start cutter)\n", spindleSpeed)
	state.Z = penUpZ

	// Iterate through paths, generating gcode for each
	transformY(svg, svg.HeightInMM())
	iterate(svg, state, xyFeedRate, zFeedRate)

	// Output gcode footer
	fmt.Println()
	fmt.Println("(end of print job)")
	fmt.Printf("G1 Z%.2f F%0.2F (Retract pen)\n", penFullRetractZ, zFeedRate)
	fmt.Println("M5 (Stop cutter)")
	fmt.Printf("G0 X%0.2F Y%0.2F F%0.2F (go home)\n", xHome, yHome, xyTravelRate)

	// For classic TimSav, return to 0,0,0 position
	//fmt.Printf("G1 Z%0.2F F%02F\n", zHeight, zFeedRate)
}

// transformY transforms the Y coordinate system from SVG's positive Y = down to Timsav's positive Y = up
func transformY(svg *cleaner.SVGXMLNode, pageHeight float64) {
	// The timSave gcode extension somehow offsets the Y values by a few mm beyond the page height.
	// TODO: this shouldn't be necessary for the final extension, but it provides a useful equivalence during development.
	//yOffset := pageHeight - 5.135

	yOffset := pageHeight

	for _, child := range svg.Children {
		for _, path := range child.Path {
			path.Y = yOffset - path.Y
			for _, drawTo := range path.DrawTo {
				drawTo.Y = yOffset - drawTo.Y
			}
		}
		transformY(child, pageHeight)
	}
}

func distance(a, b point) float64 {
	dx := a.x - b.x
	dy := a.y - b.y
	return math.Sqrt(dx*dx + dy*dy)
}

func iterate(svg *cleaner.SVGXMLNode, state *machineState, xyFeedRate, zFeedRate float64) {
	for _, child := range svg.Children {
		//fmt.Fprintf(os.Stderr, "Path %s: Style: %s Category: %d\n", child.ID, child.Styles, child.Category)
		for i, path := range child.Path {
			// This gcode comment isn't terribly useful, but for now I'm just aiming for parity with the python extension.
			fmt.Printf("\n(Object %s, path %d of %d)\n", child.ID, i+1, len(child.Path))

			dist := distance(point{x: state.X, y: state.Y}, point{x: path.X, y: path.Y})
			if dist > 0.1 { // TODO: configurable
				if state.Z <= 0 {
					//fmt.Printf("G1 Z2.00 S3400.00 (Pen Up)\n")
					fmt.Printf("G1 Z2.00 F%0.2f (Pen Up)\n", zFeedRate)
					state.Z = 2
				}
				fmt.Printf("G0 X%0.2f Y%0.2f \n", path.X, path.Y)
				state.X = path.X
				state.Y = path.Y
			}

			var z float64
			var description string
			switch child.Category {
			case cleaner.CategoryFullCut:
				z = -6.0
				description = "pen down through"
			case cleaner.CategoryScore:
				z = -4.0
				description = "pen down score"
			case cleaner.CategoryPaperCut,
				cleaner.CategoryOptional,
				cleaner.CategoryCrease:
				z = -1.0
				description = "pen down draw"
			}
			if z != state.Z {
				fmt.Printf("G1 Z%0.2f F%0.2f (%s)\n", z, zFeedRate, description)
				state.Z = z
			}

			//fmt.Printf("G0 Z%0.2f S3800.00\n", z)
			for _, drawTo := range path.DrawTo {
				switch drawTo.Command {
				case svgpath.LineTo, svgpath.ClosePath:
					fmt.Printf("G1 X%0.2f Y%0.2f F%0.2f\n", drawTo.X, drawTo.Y, xyFeedRate)
				case svgpath.CurveTo:
					fmt.Printf("G5 X%0.2f Y%0.2f I%0.2f J%0.2f P%0.2f Q%0.2f F%0.2f\n",
						drawTo.X, drawTo.Y, drawTo.X1, drawTo.Y1, drawTo.X2, drawTo.Y2, xyFeedRate)
				}
				state.X = drawTo.X
				state.Y = drawTo.Y
			}
		}
		iterate(child, state, xyFeedRate, zFeedRate)
	}
}
