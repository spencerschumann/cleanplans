//go:build float64

package float

import "math"

// Float is a floating point type. This type alias allows for easy switching between float32 and float64.
type Float = float64

func Min(a, b Float) Float {
	return math.Min(a, b)
}

func Max(a, b Float) Float {
	return math.Max(a, b)
}

func Abs(n Float) Float {
	return math.Abs(n)
}

func NaN() Float {
	return math.NaN()
}

func Inf(sign int) Float {
	return math.Inf(sign)
}
