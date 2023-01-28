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

/*func (b Blob) ToPolyline(xMajor bool) geometry.Polyline {
	var polyline geometry.Polyline
	if xMajor {
		for _, p := range b {
			polyline = append(polyline, geometry.Point{
				X: p.X,
				Y: p.Y,
			})
		}
	} else {
		for _, p := range b {
			polyline = append(polyline, geometry.Point{
				X: p.Y,
				Y: p.X,
			})
		}
	}
	return polyline
}*/

func (b Blob) ToPolyline() geometry.Polyline {
	// use linear regression to find the best-fit line to the blob
	n := 0.0
	sumX := 0.0
	sumY := 0.0
	sumXX := 0.0
	sumXY := 0.0
	minX := math.Inf(+1)
	maxX := math.Inf(-1)
	for _, run := range b {
		width := run.X2 - run.X1
		n += width
		sumY += width * (run.Y + 0.5)
		sx := width * (run.X1 + run.X2) / 2
		sumX += sx
		sumXY += sx * (run.Y + 0.5)
		sumXX += width*run.X1*run.X1 + run.X1*width*width + width*width*width/3
		minX = math.Min(minX, run.X1)
		maxX = math.Max(maxX, run.X2)
	}
	// TODO: handle vertical and near-vertical - should switch to Y as the independent variable
	betaDenominator := n*sumXX - sumX*sumX
	if betaDenominator == 0 {
		return geometry.Polyline{}
	}
	beta := (n*sumXY - sumX*sumY) / betaDenominator
	alpha := sumY/n - beta*sumX/n

	if beta < 0.5 {
		// mostly horizontal - go from minX to maxX
		return geometry.Polyline{
			{X: minX, Y: alpha + beta*minX},
			{X: maxX, Y: alpha + beta*maxX},
		}
	}

	// todo: handle mostly vertical case
	return geometry.Polyline{}
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
