package geometry

import (
	"math"
)

// TODO: consolidate other geometrical definitions into this package.

type Point struct {
	X float64
	Y float64
}

type LineSegment struct {
	A Point
	B Point
}

type Circle struct {
	Center Point
	Radius float64
}

type Arc struct {
	Start     Point
	End       Point
	Center    Point
	Clockwise bool
}

func FindArc(start, mid, end Point) Arc {
	// Calculate the coefficients for the equations of the lines through points a, b and b, c
	topLeft := start.X - mid.X
	topRight := start.Y - mid.Y
	rhsTop := (start.X*start.X - mid.X*mid.X + start.Y*start.Y - mid.Y*mid.Y) / 2.0
	bottomLeft := mid.X - end.X
	bottomRight := mid.Y - end.Y
	rhsBottom := (mid.X*mid.X - end.X*end.X + mid.Y*mid.Y - end.Y*end.Y) / 2.0

	// Calculate the x-coordinate of the center point of the arc
	originXCoeff := topLeft*bottomRight - bottomLeft*topRight
	originXCoeffValue := bottomRight*rhsTop - topRight*rhsBottom
	centerX := originXCoeffValue / originXCoeff

	// Calculate the y-coordinate of the center point of the arc
	centerY := ((start.X-mid.X)*centerX - rhsTop) / (mid.Y - start.Y)

	// Determine whether the arc should sweep clockwise or counterclockwise
	clockwise := (end.X-start.X)*(mid.Y-end.Y)-(end.Y-start.Y)*(mid.X-start.X) > 0

	return Arc{
		Start:     start,
		End:       end,
		Center:    Point{X: centerX, Y: centerY},
		Clockwise: clockwise,
	}
}

// Distance returns the distance between two points.
func (p Point) Distance(other Point) float64 {
	return math.Hypot(p.X-other.X, p.Y-other.Y)
}

// DistanceToLine returns the distance between a point and a line segment.
func (p Point) DistanceToLine(a, b Point) float64 {
	if a.X == b.X {
		return math.Abs(p.X - a.X)
	}
	if a.Y == b.Y {
		return math.Abs(p.Y - a.Y)
	}
	slope := (b.Y - a.Y) / (b.X - a.X)
	intercept := a.Y - slope*a.X
	return math.Abs(slope*p.X-p.Y+intercept) / math.Sqrt(slope*slope+1)
}

// Distance returns the distance between a point and a circle.
func (p Point) DistanceToCircle(c Circle) float64 {
	return math.Abs(math.Sqrt((p.X-c.Center.X)*(p.X-c.Center.X)+(p.Y-c.Center.Y)*(p.Y-c.Center.Y)) - c.Radius)
}

// DouglasPeucker simplifies a curve using the Douglas-Peucker algorithm
// and returns the simplified curve as a mix of line segments and circular arcs.
func DouglasPeucker(points []Point, epsilon float64) []any {
	if len(points) <= 2 {
		return []any{}
	} // TODO: 2 is ok - it's a single line; need to be careful on the recursive step to not call with < 2 points

	// find the point with the max distance from the line segment between the first and last points
	firstPoint, lastPoint := points[0], points[len(points)-1]
	dmax := 0.0
	index := 0
	maxSegLen := points[0].Distance(points[1])
	for i := 1; i < len(points)-1; i++ {
		segLen := points[i].Distance(points[i+1])
		if segLen < maxSegLen {
			maxSegLen = segLen
		}
		d := points[i].DistanceToLine(firstPoint, lastPoint)
		if d > dmax {
			index = i
			dmax = d
		}
	}

	if dmax < epsilon {
		return []any{LineSegment{A: firstPoint, B: lastPoint}}
	}

	// TODO: need another heuristic to decide if an arc should be used, to avoid transforming
	// an intentional polyline into an arc. Perhaps look at distances between successive points?
	arcLineSegLengthMax := 2.0
	if maxSegLen < arcLineSegLengthMax {
		// Check if all points are within epsilon of the circular arc defined by the first, last, and max distance points.
		arc := FindArc(firstPoint, points[index], lastPoint)
		if !math.IsNaN(arc.Center.X) && !math.IsNaN(arc.Center.Y) {
			radius := arc.Start.Distance(arc.Center)
			allWithinEpsilon := true
			for _, p := range points {
				if math.Abs(p.Distance(arc.Center)-radius) > epsilon {
					allWithinEpsilon = false
					break
				}
			}
			if allWithinEpsilon {
				return []any{arc}
			}
		}
	}

	recResults1 := DouglasPeucker(points[:index+1], epsilon)
	recResults2 := DouglasPeucker(points[index:], epsilon)
	result := make([]any, len(recResults1)+len(recResults2))
	copy(result, recResults1)
	copy(result[len(recResults1):], recResults2)
	return result
}
