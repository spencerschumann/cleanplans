package experiments

import (
	"math"
)

// Run is a horizontal set of adjacent, same-colored pixels.
// X1 is the left edge of the run of pixels, and X2 is the right edge.
// Y is the top edge of the pixels.
type Run struct {
	X1 float64
	X2 float64
	Y  float64
}

// Blob is a sequence of adjacent runs. Runs are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
// Runs is sorted by Y, with only one run per Y value.
type Blob struct {
	Runs []Run
}

type Point struct {
	X float64
	Y float64
}

type LineSegment struct {
	A     Point
	B     Point
	Width float64
}

type Rectangle struct {
	Min Point
	Max Point
}

func FindMaxRect(blob Blob) Rectangle {
	// Find the widest run and longest sequence of identical runs
	maxWidth := math.Inf(-1)
	maxI := -1
	currentRunRun := blob.Runs[0]
	currentStart := 0
	bestStart := 0
	bestEnd := 0
	for i, run := range blob.Runs {
		width := run.X2 - run.X1
		if width > maxWidth {
			maxWidth = width
			maxI = i
		}
		if currentRunRun.X1 != run.X1 || currentRunRun.X2 != run.X2 {
			runRunLen := i - currentStart
			if runRunLen > (bestEnd - bestStart) {
				bestStart = currentStart
				bestEnd = i
			}
			currentRunRun = run
		}
	}
	// Check the final run of runs
	if (len(blob.Runs) - currentStart) > (bestEnd - bestStart) {
		bestStart = currentStart
		bestEnd = len(blob.Runs)
	}

	// Grow a rectangle from the max of the widest run, or the longest sequence of identical runs
	if int(maxWidth) > (bestEnd - bestStart) {
		// Widest run won
		// Use this run as a seed to grow a "rectangular crystal" of the
		// largest dimensions that contains this run.
		maxRun := blob.Runs[maxI]
		left := maxRun.X1
		right := maxRun.X2
		seedX := (left + right) / 2
		bottom := maxRun.Y + 1
		for i := maxI + 1; i < len(blob.Runs); i++ {
			run := blob.Runs[i]
			// TODO: may want a width check in here - if we go from wide to narrow, this is probably not the same rectangle anymore
			if !(run.X1 <= seedX && seedX <= run.X2) {
				break
			}
			left = math.Max(left, run.X1)
			right = math.Min(right, run.X2)
			bottom = run.Y + 1
		}
		top := maxRun.Y
		for i := maxI - 1; i >= 0; i-- {
			run := blob.Runs[i]
			if !(run.X1 <= seedX && seedX <= run.X2) {
				break
			}
			left = math.Max(left, run.X1)
			right = math.Min(right, run.X2)
			top = run.Y
		}
		return Rectangle{
			Min: Point{
				X: left,
				Y: top,
			},
			Max: Point{
				X: right,
				Y: bottom,
			},
		}
	} else {
		// Longest sequence won
		// See if it can be grown vertically
		seqRun := blob.Runs[bestStart]
		for i := bestEnd; i < len(blob.Runs); i++ {
			run := blob.Runs[i]
			// if this run contains seqRun, let the rectangle extend through it.
			if run.X1 <= seqRun.X1 && seqRun.X2 <= run.X2 {
				bestEnd = i + 1
			}
		}
		for i := bestStart - 1; i > 0; i-- {
			run := blob.Runs[i]
			if run.X1 <= seqRun.X1 && seqRun.X2 <= run.X2 {
				bestStart = i
			}
		}
		return Rectangle{
			Min: Point{
				X: seqRun.X1,
				Y: blob.Runs[bestStart].Y,
			},
			Max: Point{
				X: seqRun.X2,
				Y: blob.Runs[bestEnd-1].Y + 1,
			},
		}
	}
}

/*func computeRunSumXX(run Run) float64 {
	a := run.X1 + 0.5
	b := run.X2 - 0.5
	return (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6
}

func computeRunSumXXX(run Run) float64 {
	a := run.X1 + 0.5
	b := run.X2 - 0.5
	return (a*a - a + b*b + b) * (a + b) * (a - b - 1) / -4
}*/
