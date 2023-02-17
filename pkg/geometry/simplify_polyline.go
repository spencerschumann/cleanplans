package geometry

import (
	"math"
)

// TODO: consolidate other geometrical definitions into this package.

type Point struct {
	X float64
	Y float64
}

type Vector2 = Point

type LineSegment struct {
	A Point
	B Point
}

type Rectangle struct {
	Min Point
	Max Point
}

type Polyline []Point

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
	// System of equations for finding the origin/center point O:
	//
	//   (A.X-B.X) * O.X + (A.Y-B.Y) * O.Y = 0.5*(A.X*A.X - B.X*B.X + A.Y*A.Y - B.Y*B.Y)
	//   (B.X-C.X) * O.X + (B.Y-C.Y) * O.Y = 0.5*(B.X*B.X - C.X*C.X + B.Y*B.Y - C.Y*C.Y)

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
	return Arc{
		Start:     start,
		End:       end,
		Center:    findCenter(start, mid, end),
		Clockwise: end.Minus(start).CrossProductZ(mid.Minus(start)) > 0,
	}
}

func (a Vector2) Minus(b Vector2) Vector2 {
	return Vector2{
		X: a.X - b.X,
		Y: a.Y - b.Y,
	}
}

func (a Vector2) Add(b Vector2) Vector2 {
	return Vector2{
		X: a.X + b.X,
		Y: a.Y + b.Y,
	}
}

func (v Vector2) Magnitude() float64 {
	return math.Hypot(v.X, v.Y)
}

func (a Vector2) CrossProductZ(b Vector2) float64 {
	return a.X*b.Y - a.Y*b.X
}

// Distance returns the distance between two points.
func (p Point) Distance(other Point) float64 {
	return math.Hypot(p.X-other.X, p.Y-other.Y)
}

// Scale returns the point scaled by the given factor f.
func (p Point) Scale(f float64) Point {
	return Point{X: p.X * f, Y: p.Y * f}
}

func (s LineSegment) Length() float64 {
	return s.A.Distance(s.B)
}

// Distance returns the distance between a point and a line segment.
func (s LineSegment) Distance(p Point) float64 {
	/*if s.A.X == s.B.X {
		return math.Abs(p.X - s.A.X)
	}
	if s.A.Y == s.B.Y {
		return math.Abs(p.Y - s.A.Y)
	}
	slope := (s.B.Y - s.A.Y) / (s.B.X - s.A.X)
	intercept := s.A.Y - slope*s.A.X
	// TODO: wait, is this right?
	return math.Abs(slope*p.X-p.Y+intercept) / math.Sqrt(slope*slope+1)*/

	// New approach, skip the above method, and do this instead.

	// Line equation, in form ax + by + c = 0
	// (y1 – y2)x + (x2 – x1)y + (x1y2 – x2y1) = 0

	// Distance to line:
	// abs(a*x0 + b*y0 + c) / sqrt(a^2 + b^2)

	/*a := s.A.Y - s.B.Y
	b := s.A.X - s.B.X
	c := s.A.X*s.B.Y - s.B.X*s.A.Y

	d := math.Abs(a*p.X+b*p.Y+c) / math.Hypot(a, b)
	return d*/

	AP := p.Minus(s.A)
	AB := s.A.Minus(s.B)
	mAP := AP.Magnitude()
	mBP := p.Minus(s.B).Magnitude()
	mAB := AB.Magnitude()

	if mAP > mAB || mBP > mAB {
		// closest point on line is outside segment boundaries, so the closest point
		// is the nearest of the two endpoints.
		return math.Min(mAP, mBP)
	}

	return math.Abs(AP.CrossProductZ(AB)) / mAB
}

// For DistanceToLine and DistanceToCircle, making the line or circle the receiver
// would remove the need for the "ToX" suffix.

// Distance returns the distance between a point and a circle.
func (p Point) DistanceToCircle(c Circle) float64 {
	return math.Abs(math.Sqrt((p.X-c.Center.X)*(p.X-c.Center.X)+(p.Y-c.Center.Y)*(p.Y-c.Center.Y)) - c.Radius)
}

func (line Polyline) EndpointDistance(p Point) float64 {
	if len(line) == 0 {
		return math.NaN()
	}
	d := line[0].Distance(p)
	if len(line) > 1 {
		d = math.Min(d, line[len(line)-1].Distance(p))
	}
	return d
}

func (line Polyline) ConnectTo(other Polyline) Polyline {
	if len(line) == 0 || len(other) == 0 {
		return nil
	}
	c1 := Polyline{line[0]}
	if len(line) > 1 {
		c1 = append(c1, line[len(line)-1])
	}
	c2 := Polyline{other[0]}
	if len(other) > 1 {
		c2 = append(c2, other[len(other)-1])
	}

	var connector Polyline
	dist := math.Inf(1)
	for _, p1 := range c1 {
		for _, p2 := range c2 {
			d := p1.Distance(p2)
			if d < dist {
				dist = d
				connector = Polyline{p1, p2}
			}
		}
	}
	return connector
}

// Simplify simplifies the polyline using the Douglas-Peucker algorithm
// and returns the simplified curve as a mix of line segments and circular arcs.
func (points Polyline) Simplify(epsilon float64) Polyline {
	if len(points) < 2 {
		return nil
	}

	// find the point with the max distance from the line segment between the first and last points
	firstPoint, lastPoint := points[0], points[len(points)-1]
	chord := LineSegment{A: firstPoint, B: lastPoint}
	if len(points) == 2 {
		return Polyline{firstPoint, lastPoint}
	}

	dmax := 0.0
	index := 0
	for i := 1; i < len(points)-1; i++ {
		d := chord.Distance(points[i])
		if d > dmax {
			index = i
			dmax = d
		}
	}

	if dmax < epsilon {
		return Polyline{firstPoint, lastPoint}
	}

	// TODO: need another heuristic to decide if an arc should be used, to avoid transforming
	// an intentional polyline into an arc. Perhaps look at distances between successive points?
	//
	// I think a better approach will be to just simplify to lines in this first step, and then
	// go back and look for arcs in an additional step.
	/*
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
	*/

	// note: need to be careful on the recursive step to not call with < 2 points
	recResults1 := Polyline(points[:index+1]).Simplify(epsilon)
	recResults2 := Polyline(points[index:]).Simplify(epsilon)

	// TODO: not sure if the direct append is actually safe, and may need to allocate
	// instead...but it may be even better to completely avoid the allocation and just
	// modify the original input slice instead, and just pass indices around and copy
	// elements as needed to consolidate.
	/*result := make([]any, len(recResults1)+len(recResults2))
	copy(result, recResults1)
	copy(result[len(recResults1):], recResults2)
	return result*/

	return append(recResults1[:len(recResults1)-1], recResults2...)
}
