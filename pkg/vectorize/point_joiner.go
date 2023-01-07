package vectorize

import (
	"math"
)

// JoinerLinePoint is a single point in a potential line.
type JoinerLinePoint struct {
	Major float32
	Minor int
	Width int
}

// JoinerLine is a sequence of adjacent points. LinePoints are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type JoinerLine []JoinerLinePoint

type PointJoiner struct {
	minor      int
	bucketSize int
	buckets    [][]JoinerLine
	lines      []JoinerLine
}

func NewPointJoiner(bucketSize, maxMajor int) *PointJoiner {
	numBuckets := maxMajor / bucketSize
	if maxMajor%bucketSize > 0 {
		numBuckets++
	}
	return &PointJoiner{
		bucketSize: bucketSize,
		buckets:    make([][]JoinerLine, numBuckets),
	}
}

func (pj *PointJoiner) NextMinor() {
	pj.minor++

	// Clear out old lines that have ended, and move them to the lines slice.
	// Otherwise the buckets will continue grow beyond their expected small size
	// and we won't get the desired O(1) performance.
	for bucketIdx, bucket := range pj.buckets {
		for i := 0; i < len(bucket); i++ {
			line := bucket[i]
			lastPoint := line[len(line)-1]
			if lastPoint.Minor < pj.minor-1 {
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

func (pj *PointJoiner) AddRun(major float32, width int) {
	// Find the appropriate bucket for the point
	pointBucketIdx := int(major / float32(pj.bucketSize))

	// Check the point's bucket, and the buckets adjacent to it.
	for bucketIdx := pointBucketIdx - 1; bucketIdx <= pointBucketIdx+1 && bucketIdx < len(pj.buckets); bucketIdx++ {
		if bucketIdx < 0 {
			continue
		}

		// Check if the point can be added to any of the existing lines in the bucket
		for i, line := range pj.buckets[bucketIdx] {
			lastPoint := line[len(line)-1]
			if math.Abs(float64(major-lastPoint.Major)) <= 1 {
				pj.buckets[bucketIdx][i] = append(line, JoinerLinePoint{Major: major, Minor: pj.minor})
				return
			}
		}
	}

	// If the point can't be added to any of the existing lines, create a new line in the bucket
	pj.buckets[pointBucketIdx] = append(pj.buckets[pointBucketIdx], JoinerLine{{Major: major, Minor: pj.minor}})
}

func (pj *PointJoiner) JoinerLines() []JoinerLine {
	// Advance Y twice to flush out all buckets
	pj.NextMinor()
	pj.NextMinor()

	return pj.lines

	// Note: should prevent any further calls to NextMinor() or AddPoint()
}
