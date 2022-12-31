package svgpath

import (
	"log"
	"math"
)

type Matrix struct {
	A float64
	B float64
	C float64
	D float64
	E float64
	F float64
}

func ParseTransform(transform string) Matrix {
	m := Matrix{
		A: 1, C: 0, E: 0,
		B: 0, D: 1, F: 0,
	}

	if transform == "" {
		return m
	}

	functions, err := ParseFunctions(transform)
	if err != nil {
		// TODO: wrong way to handle this error, return it instead
		log.Fatalf("failed to parse transform: %s", err)
	}

	for _, function := range functions {
		switch function.Name {
		case "matrix":
			if len(function.Args) != 6 {
				log.Fatalf("6 args required for matrix transform, got %v", function.Args)
			}
			m = m.Multiply(Matrix{
				A: function.Args[0], C: function.Args[2], E: function.Args[4],
				B: function.Args[1], D: function.Args[3], F: function.Args[5],
			})
		case "translate":
			if len(function.Args) != 2 && len(function.Args) != 1 {
				log.Fatalf("1 or 2 args required for translate transform, got %v", function.Args)
			}
			x := function.Args[0]
			y := 0.0
			if len(function.Args) == 2 {
				y = function.Args[1]
			}
			m = m.Multiply(Matrix{
				A: 1, C: 0, E: x,
				B: 0, D: 1, F: y,
			})
		case "scale":
			if len(function.Args) != 2 && len(function.Args) != 1 {
				log.Fatalf("1 or 2 args required for scale transform, got %v", function.Args)
			}
			x := function.Args[0]
			y := 0.0
			if len(function.Args) == 2 {
				y = function.Args[1]
			}
			m = m.Multiply(Matrix{
				A: x, C: 0, E: 0,
				B: 0, D: y, F: 0,
			})
		case "rotate":
			//  ⎡ cos(θ)  −sin(θ)  −x⋅cos(θ)+y⋅sin(θ)+x ⎤
			//  ⎢ sin(θ)   cos(θ)  −x⋅sin(θ)−y⋅cos(θ)+y |
			//  ⎣   0        0               1          ⎦
			if len(function.Args) != 3 {
				log.Fatalf("3 args required for rotate transform, got %v", function.Args)
			}
			cos := math.Cos(function.Args[0] * math.Pi / 180)
			sin := math.Sin(function.Args[0] * math.Pi / 180)
			x, y := function.Args[1], function.Args[2]
			m = m.Multiply(Matrix{
				A: cos, C: -sin, E: -x*cos + y*sin + x,
				B: sin, D: cos, F: -x*sin - y*cos + y,
			})
		default:
			log.Fatalf("unknown transform function %q %v", function.Name, function.Args)
		}
	}

	return m
}

func (m Matrix) Multiply(other Matrix) Matrix {
	return Matrix{
		A: m.A*other.A + m.C*other.B,
		B: m.B*other.A + m.D*other.B,
		C: m.A*other.C + m.C*other.D,
		D: m.B*other.C + m.D*other.D,
		E: m.A*other.E + m.C*other.F + m.E,
		F: m.B*other.E + m.D*other.F + m.F,
	}
}

func (m Matrix) transformX(x, y float64) float64 {
	return m.A*x + m.C*y + m.E
}

func (m Matrix) transformY(x, y float64) float64 {
	return m.B*x + m.D*y + m.F
}

func (m Matrix) TransformPoint(x, y float64) (float64, float64) {
	return m.transformX(x, y), m.transformY(x, y)
}

func (m Matrix) TransformPath(path []*SubPath) {
	for _, group := range path {
		group.X, group.Y = m.TransformPoint(group.X, group.Y)
		for _, drawTo := range group.DrawTo {
			drawTo.X, drawTo.Y = m.TransformPoint(drawTo.X, drawTo.Y)
			if drawTo.Command == CurveTo {
				drawTo.X1, drawTo.Y1 = m.TransformPoint(drawTo.X1, drawTo.Y1)
				drawTo.X2, drawTo.Y2 = m.TransformPoint(drawTo.X2, drawTo.Y2)
			}
		}
	}
}
