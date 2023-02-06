package vectorize

import (
	"cleanplans/pkg/geometry"
	"math"
)

// Transpose flips the X/Y coordinates in the runs, creating vertical runs from horizontal runs.
func Transpose(blob *Blob) []*Blob {
	minX := math.Inf(1)
	maxX := math.Inf(-1)
	for _, run := range blob.Runs {
		minX = math.Min(run.X1, minX)
		maxX = math.Max(run.X2, maxX)
	}

	width := maxX - minX
	tRuns := make([][]Run, int(width))
	/*for i := range tRuns {
		tRuns[i].Y = float64(i) + minX
	}*/

	add := func(x1, x2, y float64) {
		for i := int(x1 - minX); i < int(x2-minX); i++ {
			// TODO: will need an additional run if there has already been a closed run at this position.
			tRuns[i] = append(tRuns[i], Run{
				X1: y,
				X2: y,
				Y:  float64(i) + minX,
			})
		}
	}

	remove := func(x1, x2, y float64) {
		for i := int(x1 - minX); i < int(x2-minX); i++ {
			tRuns[i][len(tRuns[i])-1].X2 = y
		}
	}

	lastRun := Run{X1: minX, X2: minX}
	for _, run := range blob.Runs {
		// wherever run reaches that lastRun did not, need to activate a new tRun
		// wherever lastRun reaches that run does not, need to deactivate tRun
		remove(lastRun.X1, math.Min(lastRun.X2, run.X1), run.Y) //ok
		remove(math.Max(lastRun.X1, run.X2), lastRun.X2, run.Y) //ok
		add(run.X1, math.Min(run.X2, lastRun.X1), run.Y)        //ok
		add(math.Max(run.X1, lastRun.X2), run.X2, run.Y)        //ok
		lastRun = run
	}
	remove(lastRun.X1, lastRun.X2, lastRun.Y+1)

	var blobs []*Blob
	for _, row := range tRuns {
		for _, run := range row {
			// could use BlobFinder here, but it would need some rework,
			// and this is a simpler case - not likely to have more than 2 blobs.

			// search for a blob to extend
			found := false
			for _, blob := range blobs {
				lastRun := blob.Runs[len(blob.Runs)-1]
				if (lastRun.Y+1 == run.Y) && lastRun.overlap(run) {
					blob.Runs = append(blob.Runs, run)
					found = true
					break
				}
			}
			if !found {
				newBlob := &Blob{
					Runs:       []Run{run},
					Transposed: !blob.Transposed,
				}
				blobs = append(blobs, newBlob)
			}
		}
	}

	return blobs
}

func FindMaxRect(blob *Blob) geometry.Rectangle {
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
			currentStart = i
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
		return geometry.Rectangle{
			Min: geometry.Point{
				X: left,
				Y: top,
			},
			Max: geometry.Point{
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
		for i := bestStart - 1; i >= 0; i-- {
			run := blob.Runs[i]
			if run.X1 <= seqRun.X1 && seqRun.X2 <= run.X2 {
				bestStart = i
			}
		}
		return geometry.Rectangle{
			Min: geometry.Point{
				X: seqRun.X1,
				Y: blob.Runs[bestStart].Y,
			},
			Max: geometry.Point{
				X: seqRun.X2,
				Y: blob.Runs[bestEnd-1].Y + 1,
			},
		}
	}
}
