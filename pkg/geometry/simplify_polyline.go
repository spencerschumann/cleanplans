package geometry

import (
	"fmt"
	"math"
)

// TODO: consolidate other geometrical definitions into this package.

type Point struct {
	X float64
	Y float64
}

type Vector = Point

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

func findCenter(A, B, C Point) Point {
	// See https://math.stackexchange.com/a/1114321.
	// There are 3 equations that can be written for this,
	// where A, B, and C are chosen from the 3 input points,
	// and we're trying to find the origin point O:
	//
	// (A.X-B.X) * O.X + (A.Y-B.Y) * O.Y = 0.5*(A.X*A.X - B.X*B.X + A.Y*A.Y - B.Y*B.Y)
	// (B.X-C.X) * O.X + (B.Y-C.Y) * O.Y = 0.5*(B.X*B.X - C.X*C.X + B.Y*B.Y - C.Y*C.Y)
	// (C.X-A.X) * O.X + (C.Y-A.Y) * O.Y = 0.5*(C.X*C.X - A.X*A.X + C.Y*C.Y - A.Y*A.Y)
	//
	// Normally only two of the three are needed to find O, unless the
	// determinant of the corresponding matrix is 0 in which case a different
	// selection of equations is needed.

	determinant := func(a, b, c, d float64) float64 {
		return a*d - b*c
	}

	ab := A.Minus(B)
	bc := B.Minus(C)

	// coefficient matrix a b c d
	a, b, c, d := ab.X, ab.Y, bc.X, bc.Y
	det := determinant(a, b, c, d)
	if det == 0 {
		return Point{
			X: math.NaN(),
			Y: math.NaN(),
		}
	}

	// invert the matrix
	a, b, c, d = d, -b, -c, a

	// 2x1 matrix of constants, as a Point
	constants := Point{
		X: (A.X*A.X - B.X*B.X + A.Y*A.Y - B.Y*B.Y) / 2,
		Y: (B.X*B.X - C.X*C.X + B.Y*B.Y - C.Y*C.Y) / 2,
	}

	// multiply the matrix and vector and divide by det to find the solution
	return Point{
		X: (a*constants.X + b*constants.Y) / det,
		Y: (c*constants.X + d*constants.Y) / det,
	}
}

func FindArc(start, mid, end Point) Arc {
	center := findCenter(start, mid, end)

	// Determine whether the arc should sweep clockwise or counterclockwise
	clockwise := end.Minus(start).CrossProductZ(mid.Minus(start)) > 0

	fmt.Printf("FindArc(s=%#v, m=%#v, e=%#v) => c=%#v, cw=%t\n",
		start, mid, end, center, clockwise)

	return Arc{
		Start:     start,
		End:       end,
		Center:    center,
		Clockwise: clockwise,
	}
}

func (a Vector) Minus(b Vector) Vector {
	return Vector{
		X: a.X - b.X,
		Y: a.Y - b.Y,
	}
}

func (v Vector) Magnitude() float64 {
	return math.Hypot(v.X, v.Y)
}

func (a Vector) CrossProductZ(b Vector) float64 {
	return a.X*b.Y - a.Y*b.X
}

// Distance returns the distance between two points.
func (p Point) Distance(other Point) float64 {
	return math.Hypot(p.X-other.X, p.Y-other.Y)
}

// Distance returns the distance between a point and a line segment.
func (s LineSegment) Distance(p Point) float64 {
	if s.A.X == s.B.X {
		return math.Abs(p.X - s.A.X)
	}
	if s.A.Y == s.B.Y {
		return math.Abs(p.Y - s.A.Y)
	}
	slope := (s.B.Y - s.A.Y) / (s.B.X - s.A.X)
	intercept := s.A.Y - slope*s.A.X
	// TODO: wait, is this right?
	return math.Abs(slope*p.X-p.Y+intercept) / math.Sqrt(slope*slope+1)
}

// For DistanceToLine and DistanceToCircle, making the line or circle the receiver
// would remove the need for the "ToX" suffix.

// Distance returns the distance between a point and a circle.
func (p Point) DistanceToCircle(c Circle) float64 {
	return math.Abs(math.Sqrt((p.X-c.Center.X)*(p.X-c.Center.X)+(p.Y-c.Center.Y)*(p.Y-c.Center.Y)) - c.Radius)
}

type Step struct {
	Name     string
	Points   Polyline
	Chord    LineSegment
	FarPoint Point
	Arc      Arc
	Result   []any
	Message  string
}

type Polyline []Point

// Simplify simplifies the polyline using the Douglas-Peucker algorithm
// and returns the simplified curve as a mix of line segments and circular arcs.
func (points Polyline) Simplify(epsilon float64, steps chan<- Step) []any {
	if len(points) < 2 {
		return []any{}
	}

	addStep := func(step Step) {
		if steps == nil {
			return
		}
		steps <- step
	}

	addStep(Step{
		Name:   "simplify",
		Points: points,
	})

	// find the point with the max distance from the line segment between the first and last points
	firstPoint, lastPoint := points[0], points[len(points)-1]
	chord := LineSegment{A: firstPoint, B: lastPoint}

	addStep(Step{
		Name:   "chord",
		Points: points,
		Chord:  chord,
	})

	if len(points) == 2 {
		addStep(Step{
			Name:   "result",
			Points: points,
			Result: []any{chord},
		})
		return []any{chord}
	}

	dmax := 0.0
	index := 0
	maxSegLen := points[0].Distance(points[1])
	for i := 1; i < len(points)-1; i++ {
		segLen := points[i].Distance(points[i+1])
		if segLen < maxSegLen {
			maxSegLen = segLen
		}
		d := chord.Distance(points[i])
		if d > dmax {
			index = i
			dmax = d
		}
	}
	addStep(Step{
		Name:     "maxDist",
		Points:   points,
		Chord:    chord,
		FarPoint: points[index],
	})

	if dmax < epsilon {
		addStep(Step{
			Name:   "result",
			Points: points,
			Result: []any{chord},
		})
		return []any{chord}
	}

	// TODO: need another heuristic to decide if an arc should be used, to avoid transforming
	// an intentional polyline into an arc. Perhaps look at distances between successive points?
	arcLineSegLengthMax := 2.0
	if maxSegLen < arcLineSegLengthMax {
		// Check if all points are within epsilon of the circular arc defined by the first, last, and max distance points.
		arc := FindArc(firstPoint, points[index], lastPoint)
		addStep(Step{
			Name:   "findArc",
			Points: points,
			Arc:    arc,
		})
		if !math.IsNaN(arc.Center.X) && !math.IsNaN(arc.Center.Y) {
			radius := arc.Start.Distance(arc.Center)
			allWithinEpsilon := true
			for _, p := range points {
				if math.Abs(p.Distance(arc.Center)-radius) > epsilon {
					addStep(Step{
						Name:     "arcDistanceExceeded",
						Points:   points,
						Arc:      arc,
						FarPoint: p,
					})
					allWithinEpsilon = false
					break
				}
			}
			if allWithinEpsilon {
				addStep(Step{
					Name:   "result",
					Points: points,
					Result: []any{arc},
				})
				return []any{arc}
			}
		}
	}

	// note: need to be careful on the recursive step to not call with < 2 points
	recResults1 := Polyline(points[:index+1]).Simplify(epsilon, steps)
	recResults2 := Polyline(points[index:]).Simplify(epsilon, steps)
	result := make([]any, len(recResults1)+len(recResults2))
	copy(result, recResults1)
	copy(result[len(recResults1):], recResults2)
	addStep(Step{
		Name:   "result",
		Points: points,
		Result: result,
	})
	return result
}
