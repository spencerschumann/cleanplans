package vectorize

import (
	"math"
)

// LinePoint is a single point in a potential line.
type LinePoint struct {
	X     float32
	Y     int
	Width int
}

// Line is a sequence of adjacent points. LinePoints are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type Line []LinePoint

type PointJoiner struct {
	y          int
	bucketSize int
	buckets    [][]Line
	lines      []Line
}

func NewPointJoiner(bucketSize, maxX int) *PointJoiner {
	numBuckets := maxX / bucketSize
	if maxX%bucketSize > 0 {
		numBuckets++
	}
	return &PointJoiner{
		bucketSize: bucketSize,
		buckets:    make([][]Line, numBuckets),
	}
}

func (pj *PointJoiner) NextY() {
	pj.y++

	// Clear out old lines that have ended, and move them to the lines slice.
	// Otherwise the buckets will continue grow beyond their expected small size
	// and we won't get the desired O(1) performance.
	for bucketIdx, bucket := range pj.buckets {
		for i := 0; i < len(bucket); i++ {
			line := bucket[i]
			lastPoint := line[len(line)-1]
			if lastPoint.Y < pj.y-1 {
				// Add the linen to the output lines slice, if it has at least 2 points
				if len(line) >= 2 {
					pj.lines = append(pj.lines, line)
				}
				// Remove the line from the bucket
				bucket[i] = bucket[len(bucket)-1]
				bucket = bucket[:len(bucket)-1]
				pj.buckets[bucketIdx] = bucket
				i--
			}
		}
	}
}

func (pj *PointJoiner) AddPoint(x float32) {
	// Find the appropriate bucket for the point
	pointBucketIdx := int(x / float32(pj.bucketSize))

	// Check the point's bucket, and the buckets adjacent to it.
	for bucketIdx := pointBucketIdx - 1; bucketIdx <= pointBucketIdx+1 && bucketIdx < len(pj.buckets); bucketIdx++ {
		if bucketIdx < 0 {
			continue
		}

		// Check if the point can be added to any of the existing lines in the bucket
		for i, line := range pj.buckets[bucketIdx] {
			lastPoint := line[len(line)-1]
			if math.Abs(float64(x-lastPoint.X)) <= 1 {
				pj.buckets[bucketIdx][i] = append(line, LinePoint{X: x, Y: pj.y})
				return
			}
		}
	}

	// If the point can't be added to any of the existing lines, create a new line in the bucket
	pj.buckets[pointBucketIdx] = append(pj.buckets[pointBucketIdx], Line{{X: x, Y: pj.y}})
}

func (pj *PointJoiner) Lines() []Line {
	// Advance Y twice to flush out all buckets
	pj.NextY()
	pj.NextY()

	return pj.lines

	// Note: should prevent any further calls to NextY() or AddPoint()
}
