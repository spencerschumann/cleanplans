package vectorize

import (
	"cleanplans/pkg/geometry"
	"math"
)

// Run is a horizontal set of adjacent, same-colored pixels.
type Run struct {
	X     float32
	Y     float32
	Width int
}

// Blob is a sequence of adjacent runs. Runs are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type Blob []Run

func (b Blob) ToPolyline(xMajor bool) geometry.Polyline {
	var polyline geometry.Polyline
	if xMajor {
		for _, p := range b {
			polyline = append(polyline, geometry.Point{
				X: float64(p.X),
				Y: float64(p.Y),
			})
		}
	} else {
		for _, p := range b {
			polyline = append(polyline, geometry.Point{
				X: float64(p.Y),
				Y: float64(p.X),
			})
		}
	}
	return polyline
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
	return math.Abs(float64(r.X-other.X)) <= float64(r.Width+other.Width)/2
}

func (bf *BlobFinder) runBuckets(x float32, width int) (int, int) {
	// Convert x & width to run endpoints
	x1 := x - float32(width)/2
	x2 := x + float32(width)/2

	// diagonally adjacent runs are treated as connected, so extend the x values by 1 to compensate.
	x1++
	x2--

	first := int(x1) / bf.bucketSize
	last := int(x2) / bf.bucketSize

	if first < 0 {
		first = 0
	}
	if last >= len(bf.buckets) {
		last = len(bf.buckets) - 1
	}

	return first, last
}

// TODO: after switching from "point joiner" to "blob finder", it may be better to represent runs by their first x coordinate rather than the midpoint.
func (bf *BlobFinder) AddRun(x float32, width int) {
	run := Run{X: x, Y: float32(bf.y), Width: width}
	firstBucketIdx, lastBucketIdx := bf.runBuckets(x, width)
	for i := firstBucketIdx; i <= lastBucketIdx; i++ {
		// Check if the run can be added to any of the existing blobs in the bucket
		for key, blob := range bf.buckets[i] {
			prevRun := (*blob)[len(*blob)-1]
			if !run.overlap(prevRun) {
				continue
			}

			*blob = append(*blob, run)

			prevFirst, prevLast := bf.runBuckets(prevRun.X, prevRun.Width)

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

				// For remaining buckets covered by both or neither, no change is needed.
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

	return pj.blobs

	// Note: should prevent any further calls to NextMinor() or AddPoint()
}
