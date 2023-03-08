//go:build !float64

package float

import "github.com/chewxy/math32"

// Float is a floating point type. This type alias allows for easy switching between float32 and float64.
type Float = float32

func Min(a, b Float) Float {
	return math32.Min(a, b)
}

func Max(a, b Float) Float {
	return math32.Max(a, b)
}

func Abs(n Float) Float {
	return math32.Abs(n)
}

func NaN() Float {
	return math32.NaN()
}

func Inf(sign int) Float {
	return math32.Inf(sign)
}
