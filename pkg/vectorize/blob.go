package vectorize

import (
	"cleanplans/pkg/geometry"
	"fmt"
	"sort"
	"time"

	"github.com/chewxy/math32"
)

// Outline creates a polyline that traces the outline of the blob.
func (blob *Blob) Outline(margin Float, transposed bool) geometry.Polyline {
	x1, x2 := blob.Runs[0].X1, blob.Runs[0].X2
	left := geometry.Polyline{{X: x1 + margin, Y: blob.Runs[0].Y + margin}}
	right := geometry.Polyline{{X: x2 - margin, Y: blob.Runs[0].Y + margin}}
	lastRun := blob.Runs[0]
	for _, run := range blob.Runs[1:] {
		if run.X1 < x1 {
			left = append(left, geometry.Point{X: x1 + margin, Y: run.Y + margin})
			left = append(left, geometry.Point{X: run.X1 + margin, Y: run.Y + margin})
		} else if x1 < run.X1 {
			left = append(left, geometry.Point{X: x1 + margin, Y: lastRun.Y + 1 - margin})
			left = append(left, geometry.Point{X: run.X1 + margin, Y: lastRun.Y + 1 - margin})
		}
		if run.X2 < x2 {
			right = append(right, geometry.Point{X: x2 - margin, Y: lastRun.Y + 1 - margin})
			right = append(right, geometry.Point{X: run.X2 - margin, Y: lastRun.Y + 1 - margin})
		} else if x2 < run.X2 {
			right = append(right, geometry.Point{X: x2 - margin, Y: run.Y + margin})
			right = append(right, geometry.Point{X: run.X2 - margin, Y: run.Y + margin})
		}
		x1 = run.X1
		x2 = run.X2
		lastRun = run
	}
	left = append(left, geometry.Point{X: lastRun.X1 + margin, Y: lastRun.Y + 1 - margin})
	right = append(right, geometry.Point{X: lastRun.X2 - margin, Y: lastRun.Y + 1 - margin})
	line := geometry.Polyline(left)
	reverse(right)
	line = append(line, right...)
	// Close the loop
	line = append(line, line[0])

	if transposed {
		for i := range line {
			p := &line[i]
			p.X, p.Y = p.Y, p.X
		}
	}
	return line
}

// Transpose flips the X/Y coordinates in the blobs, creating vertical runs from horizontal runs.
func Transpose(blobs []*Blob, maxX, maxY int) []Runs {

	// TODO: idea, may not need to actually build blobs until later on. Instead I can
	// use this new idea of [][]Run. I could even apply the bucketization idea for efficient
	// lookups. Then I can process the horizontal and vertical sets of runs to sort out
	// which to use for each pixel, and only then build up the blobs.
	// Update: wrong, the alg used here needs the runs to have been pre-arranged into blobs.

	t1 := time.Now()
	tRuns := make([]Runs, maxX)

	prealloc := make([]Run, 100)
	prealloc_i := -1
	makeRun := func(x1, x2, y Float) *Run {
		prealloc_i++
		if prealloc_i >= len(prealloc) {
			prealloc = make([]Run, 100)
			prealloc_i = 0
		}

		run := prealloc[prealloc_i]
		run.X1 = x1
		run.X2 = x2
		run.Y = y
		return &run
	}

	beginRun := func(x1, x2, y Float) {
		for i := int(x1); i < int(x2); i++ {
			if tRuns[i] == nil {
				tRuns[i] = make(Runs, 0, 50)
			}
			tRuns[i] = append(tRuns[i], makeRun(y, y, Float(i)))
		}
	}

	endRun := func(x1, x2, y Float) {
		for i := int(x1); i < int(x2); i++ {
			tRuns[i][len(tRuns[i])-1].X2 = y
		}
	}

	for _, blob := range blobs {
		lastRun := Run{}
		for _, run := range blob.Runs {
			// wherever lastRun reaches that run does not, need to deactivate tRun
			endRun(lastRun.X1, math32.Min(lastRun.X2, run.X1), run.Y)
			endRun(math32.Max(lastRun.X1, run.X2), lastRun.X2, run.Y)
			// wherever run reaches that lastRun did not, need to activate a new tRun
			beginRun(run.X1, math32.Min(run.X2, lastRun.X1), run.Y)
			beginRun(math32.Max(run.X1, lastRun.X2), run.X2, run.Y)
			lastRun = *run
		}
		endRun(lastRun.X1, lastRun.X2, lastRun.Y+1)
	}
	t2 := time.Now()
	fmt.Printf("  **time to construct tRuns: %gms\n", timeDeltaMS(t1, t2))

	for rowIndex, row := range tRuns {
		sort.Slice(row, func(i, j int) bool {
			return row[i].X1 < row[j].X1
		})
		j := 0
		k := 0
		for j < len(row) {
			run := row[j]
			j++
			// coalesce adjacent runs
			for j < len(row) && run.X2 == row[j].X1 {
				run.X2 = row[j].X2
				j++
			}
			row[k] = run
			k++
		}
		tRuns[rowIndex] = row[:k]
	}
	t3 := time.Now()
	fmt.Printf("  *time to process tRuns: %gms\n", timeDeltaMS(t2, t3))

	return tRuns
}

func FindMaxRect(blob *Blob) geometry.Rectangle {
	// Find the widest run and longest sequence of identical runs
	maxWidth := math32.Inf(-1)
	maxI := -1
	currentRunRun := blob.Runs[0]
	var currentStart, bestStart, bestEnd int
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
			left = math32.Max(left, run.X1)
			right = math32.Min(right, run.X2)
			bottom = run.Y + 1
		}
		top := maxRun.Y
		for i := maxI - 1; i >= 0; i-- {
			run := blob.Runs[i]
			if !(run.X1 <= seedX && seedX <= run.X2) {
				break
			}
			left = math32.Max(left, run.X1)
			right = math32.Min(right, run.X2)
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
