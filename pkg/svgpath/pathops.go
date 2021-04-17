package svgpath

import "math"

func (path *SubPath) StartPoint() (float64, float64) {
	return path.X, path.Y
}

func (path *SubPath) EndPoint() (float64, float64) {
	if len(path.DrawTo) > 0 {
		last := path.DrawTo[len(path.DrawTo)-1]
		return last.X, last.Y
	}
	return path.X, path.Y
}

// Reverse reverses a path and returns the result
func (path *SubPath) Reverse() *SubPath {
	reversed := &SubPath{}
	reversed.X, reversed.Y = path.EndPoint()
	for i := len(path.DrawTo) - 1; i >= 0; i-- {
		drawTo := path.DrawTo[i]
		var prevX, prevY float64
		if i > 0 {
			prevX = path.DrawTo[i-1].X
			prevY = path.DrawTo[i-1].Y
		} else {
			prevX = path.X
			prevY = path.Y
		}
		rDrawTo := &DrawTo{
			Command: drawTo.Command,
			X:       prevX,
			Y:       prevY,
		}
		reversed.DrawTo = append(reversed.DrawTo, rDrawTo)
		switch drawTo.Command {
		case LineTo, ClosePath:
			// Nothing more to do
		case CurveTo:
			rDrawTo.X1 = drawTo.X2
			rDrawTo.Y1 = drawTo.Y2
			rDrawTo.X2 = drawTo.X1
			rDrawTo.Y2 = drawTo.Y1
		}
	}
	return reversed
}

func pointDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

func (path *SubPath) simplifyCurves() {
	// Convert linear curves to lines
	lastX, lastY := path.StartPoint()
	for _, drawTo := range path.DrawTo {
		if drawTo.Command == CurveTo {
			// Ensure both control points are close enough to the line segment,
			// and that they lie within the line segment.
			dx := drawTo.X - lastX
			dy := drawTo.Y - lastY
			dist1 := math.Abs(dx*(lastY-drawTo.Y1)-dy*(lastX-drawTo.X1)) /
				math.Sqrt(dx*dx+dy*dy)
			dist2 := math.Abs(dx*(lastY-drawTo.Y2)-dy*(lastX-drawTo.X2)) /
				math.Sqrt(dx*dx+dy*dy)
			length := pointDistance(lastX, lastY, drawTo.X, drawTo.Y) + 0.02
			if dist1 < 0.02 && dist2 < 0.02 &&
				pointDistance(lastX, lastY, drawTo.X1, drawTo.Y1) < length &&
				pointDistance(drawTo.X, drawTo.Y, drawTo.X1, drawTo.Y1) < length &&
				pointDistance(lastX, lastY, drawTo.X2, drawTo.Y2) < length &&
				pointDistance(drawTo.X, drawTo.Y, drawTo.X2, drawTo.Y2) < length {
				drawTo.Command = LineTo
			}
		}
		lastX, lastY = drawTo.X, drawTo.Y
	}
}

func (path *SubPath) simplifyLines() {
	// Remove redundant points along line segments
	lastX, lastY := path.StartPoint()
	keepIndex := 0
	for i, drawTo := range path.DrawTo {
		if i == len(path.DrawTo)-1 {
			path.DrawTo[keepIndex] = drawTo
			keepIndex++
			break
		}
		next := path.DrawTo[i+1]
		// Get the distance between this point and the line segment between "last" and "next".
		dx := next.X - lastX
		dy := next.Y - lastY
		dist := math.Abs(dx*(lastY-drawTo.Y)-dy*(lastX-drawTo.X)) /
			math.Sqrt(dx*dx+dy*dy)

		// Only keep the point if it's needed
		if drawTo.Command != LineTo || next.Command != LineTo || dist > 0.01 { // TODO: configurable
			path.DrawTo[keepIndex] = drawTo
			keepIndex++
			lastX, lastY = drawTo.X, drawTo.Y
		}
	}
	path.DrawTo = path.DrawTo[:keepIndex]
}

// Simplify simplifies a path
func (path *SubPath) Simplify() {
	path.simplifyCurves()
	path.simplifyLines()
}
