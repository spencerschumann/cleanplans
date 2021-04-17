package cleaner

import (
	"cleanplans/pkg/svgpath"
	"fmt"
	"log"
	"math"
	"regexp"
)

// TODO: why do the functions in this file deserve to be in the file
// that bears the package's name?

func shouldKeepPath(path []*svgpath.SubPath) bool {
	if len(path) == 0 {
		return false
	}

	// Get rid of dots - lines that start and stop at the same point.
	// These are used for the magenta "remove paper" speckles that really bog inkscape down.
	// For now, just check for a path with a single dot; but note that there could be sub-path speckles that should also get culled from non-speckle parts of the path.
	for _, group := range path {
		if len(group.DrawTo) == 1 && group.DrawTo[0].Command == svgpath.LineTo &&
			group.X == group.DrawTo[0].X && group.Y == group.DrawTo[0].Y {
			return false
		}
		// TODO: could also have speckles caused by commands other than LineTo
		// And could have speckles with multiple repetitions of the single point
	}

	// Keep everything else
	return true
}

func shouldKeepFill(fill string) bool {
	return fill == "none" || fill == ""
}

func scaleFactor(baseUnits string) float64 {
	/*
		Units as defined at https://www.w3.org/TR/css3-values/#absolute-lengths

		unit	name	equivalence
		cm	centimeters	1cm = 96px/2.54
		mm	millimeters	1mm = 1/10th of 1 cm
		Q	quarter-millimeters	1Q = 1/40th of 1 cm
		in	inches	1 in = 2.54cm = 96px
		pc	picas	1 pc = 1/6th of 1 in
		pt	points	1 pt = 1/72th of 1 in
		px	pixels	1 px = 1/96th of 1 in
	*/
	factors := map[string]float64{
		"cm": 10,
		"mm": 1,
		"Q":  0.25,
		"in": 25.4,
		"pc": 25.4 / 6,
		"pt": 25.4 / 72,
		"px": 25.4 / 96,
	}
	if factor, ok := factors[baseUnits]; ok {
		return factor
	}
	// Default to no scaling
	return 1
}

func scaleToMM(value float64, baseUnits string) float64 {
	return scaleFactor(baseUnits) * value
}

// TODO: filter as one step, then absolute MM as another? Except filtering requires processing the path...
func (svg *SVGXMLNode) FilteredAbsoluteMM() {
	svg.findBaseUnits()

	var descend func(node *SVGXMLNode, matrix svgpath.Matrix)
	descend = func(node *SVGXMLNode, matrix svgpath.Matrix) {
		matrix = matrix.Multiply(svgpath.ParseTransform(node.Transform))
		node.Transform = ""

		path, err := svgpath.Parse(node.D)
		if err != nil {
			// TODO: wrong way to handle these errors
			log.Fatalf("failed to parse path: %s", err)
		}
		node.path = path

		node.category = classifyStroke(node.Style("stroke"))

		if shouldKeepFill(node.Style("fill")) && node.category != CategoryNone && shouldKeepPath(path) {
			// Scale the stroke width appropriately
			width := 0.0
			strokeWidth := node.Style("stroke-width")
			if strokeWidth != "" {
				width = ParseNumber(strokeWidth)
				x1, y1 := matrix.TransformPoint(0, 0)
				x2, y2 := matrix.TransformPoint(width, 0)
				dx := x2 - x1
				dy := y2 - y1
				width = math.Sqrt(dx*dx + dy*dy)
				node.SetStyle("stroke-width", FormatNumber(width))
			}

			// Cutoff for stroke width to filter out the logo
			if width > 0.2 {
				matrix.TransformPath(path)
				svg.Children = append(svg.Children, node)
			}
		}
		for _, child := range node.Children {
			descend(child, matrix)
		}
	}

	factor := scaleFactor(svg.baseUnits)
	scaleMatrix := svgpath.Matrix{
		A: factor, C: 0, E: 0,
		B: 0, D: factor, F: 0,
	}

	oldChildren := svg.Children
	svg.Children = nil
	for _, child := range oldChildren {
		descend(child, scaleMatrix)
	}
}

// TODO: maybe do this as part of the parse function, so that we aren't left with an object in an unusable state
func (svg *SVGXMLNode) findBaseUnits() {
	// Determine base units
	// TODO: it would be better to have separate structs for marshalling and processing,
	// to eliminate the possibility of forgetting to re-marshal paths and styles
	unitsRE := regexp.MustCompile(`([0-9\.]+)([a-zA-z]+)`)
	widthMatch := unitsRE.FindStringSubmatch(svg.Width)
	heightMatch := unitsRE.FindStringSubmatch(svg.Height)
	if widthMatch == nil || heightMatch == nil {
		if (widthMatch == nil) != (heightMatch == nil) {
			log.Fatalf("units for width and height don't match!")
		}
		// If no units are provided, assume pixels
		svg.baseUnits = "px"
		svg.widthInMM = scaleToMM(ParseNumber(svg.Width), svg.baseUnits)
		svg.heightInMM = scaleToMM(ParseNumber(svg.Height), svg.baseUnits)
	} else {
		if widthMatch[2] != heightMatch[2] {
			log.Fatalf("units for width (%s) and height (%s) don't match!", widthMatch[2], heightMatch[2])
		}
		svg.baseUnits = widthMatch[2]
		svg.widthInMM = scaleToMM(ParseNumber(widthMatch[1]), svg.baseUnits)
		svg.heightInMM = scaleToMM(ParseNumber(heightMatch[1]), svg.baseUnits)
	}
}

// Now that the paths are all in absolute mm measurements, find the bounds,
// rotate, and center in the output document
func (svg *SVGXMLNode) RotateAndCenter(widthInMM, heightInMM float64) {
	// Rotate to portrait orientation if needed
	if svg.widthInMM > svg.heightInMM {
		rotate := func(x, y float64) (float64, float64) {
			// Y gets X
			// X gets Height - Y
			return svg.heightInMM - y, x
		}

		for _, node := range svg.Children {
			for _, path := range node.path {
				path.X, path.Y = rotate(path.X, path.Y)
				for _, drawTo := range path.DrawTo {
					drawTo.X, drawTo.Y = rotate(drawTo.X, drawTo.Y)
					drawTo.X1, drawTo.Y1 = rotate(drawTo.X1, drawTo.Y1)
					drawTo.X2, drawTo.Y2 = rotate(drawTo.X2, drawTo.Y2)
				}
			}
		}
	}

	svg.widthInMM = widthInMM
	svg.heightInMM = heightInMM
	svg.ViewBox = fmt.Sprintf("0 0 %f %f", svg.widthInMM, svg.heightInMM)

	minX, minY, maxX, maxY := svg.Bounds()
	cx := (maxX - minX) / 2
	cy := (maxY - minY) / 2

	dx := svg.widthInMM/2 - cx - minX
	dy := svg.heightInMM/2 - cy - minY
	translate := func(x, y float64) (float64, float64) {
		return x + dx, y + dy
	}
	for _, node := range svg.Children {
		for _, path := range node.path {
			path.X, path.Y = translate(path.X, path.Y)
			for _, drawTo := range path.DrawTo {
				drawTo.X, drawTo.Y = translate(drawTo.X, drawTo.Y)
				drawTo.X1, drawTo.Y1 = translate(drawTo.X1, drawTo.Y1)
				drawTo.X2, drawTo.Y2 = translate(drawTo.X2, drawTo.Y2)
			}
		}
	}
}
