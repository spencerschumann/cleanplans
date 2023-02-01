package vectorize

import (
	"cleanplans/pkg/geometry"
	"math"
)

// Run is a horizontal set of adjacent, same-colored pixels.
type Run struct {
	X1 float64
	X2 float64
	Y  float64
}

// Blob is a sequence of adjacent runs. Runs are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type Blob []Run

// Use the technique from https://dtcenter.org/sites/default/files/community-code/met/docs/write-ups/circle_fit.pdf
func (blob Blob) BestFitCircle() geometry.Circle {
	// Calculate centroid (average x and y coordinates) of pixels in the blob
	// Note: this was already calculated in ToPolyline; if that function is always called first,
	// the result could be used to avoid recomputing it here. Might be worth converting blob
	// to a struct and memoizing the result.
	n := 0.0
	sumX := 0.0
	sumY := 0.0
	for _, run := range blob {
		y := run.Y + 0.5
		width := run.X2 - run.X1
		n += width
		sumY += width * y
		runSumX := width * (run.X1 + run.X2) / 2
		sumX += runSumX
	}
	avgX := sumX / n
	avgY := sumY / n

	// Calculate Sxxx sums for Eq. 4 and Eq. 5
	Suv := 0.0
	Suu := 0.0
	Svv := 0.0
	Suuu := 0.0
	Svvv := 0.0
	Suvv := 0.0
	Svuu := 0.0
	for _, run := range blob {
		u1 := run.X1 - avgX
		u2 := run.X2 - avgX
		v := run.Y + 0.5 - avgY
		width := u2 - u1
		runSu := width * (u1 + u2) / 2
		Suv += runSu * v
		a := u1 + 0.5
		b := u2 - 0.5
		runSuu := (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6
		Suu += runSuu
		Svuu += runSuu * v
		runSvv := v * v * width
		Svv += runSvv
		Svvv += runSvv * v
		Suvv += runSu * v * v
		Suuu += (a*a - a + b*b + b) * (a + b) * (a - b - 1) / -4
	}

	// Now solve the system of equations Eq. 4 and Eq. 5, substituting variables to match this format:
	// 	a1 * uc + b1 * vc = c1
	// 	a2 * uc + b1 * vc = c2
	a1 := Suu
	b1 := Suv
	c1 := (Suuu + Suvv) / 2
	a2 := Suv
	b2 := Svv
	c2 := (Svvv + Svuu) / 2
	det := a1*b2 - a2*b1
	if det == 0 {
		// fail - can't find a suitable circle center
		return geometry.Circle{}
	}
	uc := (c1*b2 - c2*b1) / det
	vc := (a1*c2 - a2*c1) / det

	// Substitute uc and uv into Eq. 6 to compute radius
	radius := math.Sqrt(uc*uc + vc*vc + (Suu+Svv)/n)
	xc := uc + avgX
	yc := vc + avgY

	return geometry.Circle{
		Center: geometry.Point{X: xc, Y: yc},
		Radius: radius,
	}
}

func (blob Blob) ToPolyline() geometry.Polyline {
	// use linear regression to find the best-fit line to the blob
	n := 0.0
	Sx := 0.0
	Sxx := 0.0
	Sy := 0.0
	Syy := 0.0
	Sxy := 0.0
	minX := math.Inf(+1)
	maxX := math.Inf(-1)
	minY := math.Inf(+1)
	maxY := math.Inf(-1)
	for _, run := range blob {
		y := run.Y + 0.5
		width := run.X2 - run.X1
		n += width
		Sy += width * y
		runSumX := width * (run.X1 + run.X2) / 2
		Sx += runSumX
		Sxy += runSumX * y

		//a := run.X1 + 0.5
		//b := run.X2 - 0.5
		//Sxx += (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6

		Sxx += width*run.X1*run.X1 + run.X1*width*width + width*width*width/3

		Syy += y * y * width
		minX = math.Min(minX, run.X1)
		maxX = math.Max(maxX, run.X2)
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}

	betaDenominatorX := n*Sxx - Sx*Sx
	betaDenominatorY := n*Syy - Sy*Sy
	if betaDenominatorY < betaDenominatorX {
		// mostly horizontal line
		betaNumerator := n*Sxy - Sx*Sy
		beta := betaNumerator / betaDenominatorX
		alpha := Sy/n - beta*Sx/n
		return geometry.Polyline{
			{X: minX, Y: alpha + beta*minX},
			{X: maxX, Y: alpha + beta*maxX},
		}
	} else {
		// mostly vertical line
		betaNumerator := n*Sxy - Sx*Sy
		beta := betaNumerator / betaDenominatorY
		alpha := Sx/n - beta*Sy/n
		return geometry.Polyline{
			{Y: minY, X: alpha + beta*minY},
			{Y: maxY, X: alpha + beta*maxY},
		}
	}
}

type BlobFinder struct {
	y          int
	bucketSize int
	buckets    []map[Run]*Blob
	blobs      []Blob
}

func NewBlobFinder(bucketSize, maxX int) *BlobFinder {
	numBuckets := maxX / bucketSize
	if maxX%bucketSize > 0 {
		numBuckets++
	}
	buckets := make([]map[Run]*Blob, numBuckets)
	for i := 0; i < numBuckets; i++ {
		buckets[i] = map[Run]*Blob{}
	}
	return &BlobFinder{
		bucketSize: bucketSize,
		buckets:    buckets,
	}
}

func (bf *BlobFinder) NextY() {
	bf.y++

	// Clear out old lines that have ended, and move them to the lines slice.
	// Otherwise the buckets will continue grow beyond their expected small size
	// and we won't get the desired O(1) performance.
	ended := map[*Blob]struct{}{}
	for _, bucket := range bf.buckets {
		for key, blob := range bucket {
			lastRun := (*blob)[len(*blob)-1]
			if int(lastRun.Y) < bf.y-1 {
				ended[blob] = struct{}{}
				delete(bucket, key)
			}
		}
	}
	for blob := range ended {
		bf.blobs = append(bf.blobs, *blob)
	}
}

// overlap returns true if the two runs overlap, including diagonally
func (r Run) overlap(other Run) bool {
	return r.X1 <= other.X2 && other.X1 <= r.X2
}

func (bf *BlobFinder) runBuckets(run Run) (int, int) {
	// diagonally adjacent runs are treated as connected, so extend the x values by 1 to compensate.
	first := int(run.X1-1) / bf.bucketSize
	last := int(run.X2+1) / bf.bucketSize

	if first < 0 {
		first = 0
	}
	if last >= len(bf.buckets) {
		last = len(bf.buckets) - 1
	}

	return first, last
}

func (bf *BlobFinder) AddRun(x1, x2 float64) {
	run := Run{X1: x1, X2: x2, Y: float64(bf.y)}
	firstBucketIdx, lastBucketIdx := bf.runBuckets(run)
	for i := firstBucketIdx; i <= lastBucketIdx; i++ {
		// Check if the run can be added to any of the existing blobs in the bucket
		for key, blob := range bf.buckets[i] {
			prevRun := (*blob)[len(*blob)-1]
			if !run.overlap(prevRun) {
				continue
			}

			*blob = append(*blob, run)

			prevFirst, prevLast := bf.runBuckets(prevRun)

			start := firstBucketIdx
			if prevFirst < start {
				start = prevFirst
			}
			end := lastBucketIdx
			if end < prevLast {
				end = prevLast
			}
			for j := start; j <= end; j++ {
				inRun := firstBucketIdx <= j && j <= lastBucketIdx
				inPrev := prevFirst <= j && j <= prevLast

				if inPrev && !inRun {
					// Remove the blob from previous buckets that aren't within the new run
					delete(bf.buckets[j], key)
				}

				if !inPrev && inRun {
					// Add the blob to buckets that weren't in prev
					bf.buckets[j][key] = blob
				}

				// For remaining buckets covered by both or neither runs, no change is needed.
			}
			return
		}
	}

	// If the run can't be added to any of the existing blobs, create a new blob and add it to the buckets
	blob := Blob{run}
	for i := firstBucketIdx; i <= lastBucketIdx; i++ {
		bf.buckets[i][run] = &blob
	}
}

func (pj *BlobFinder) Blobs() []Blob {
	// Advance Y twice to flush out all buckets
	pj.NextY()
	pj.NextY()

	// Further calls to NextY or AddRun are undefined.
	pj.buckets = nil

	return pj.blobs
}
