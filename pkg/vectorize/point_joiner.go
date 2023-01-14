package vectorize

import (
	"cleanplans/pkg/cfg"
	"cleanplans/pkg/geometry"
	"math"
)

// JoinerLinePoint is a single point in a potential line.
type JoinerLinePoint struct {
	Major float32
	Minor float32
	Width int
}

// JoinerLine is a sequence of adjacent points. LinePoints are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type JoinerLine []JoinerLinePoint

func (jl JoinerLine) ToPolyline(xMajor bool) geometry.Polyline {
	var polyline geometry.Polyline
	if xMajor {
		for _, p := range jl {
			polyline = append(polyline, geometry.Point{
				X: float64(p.Major),
				Y: float64(p.Minor),
			})
		}
	} else {
		for _, p := range jl {
			polyline = append(polyline, geometry.Point{
				X: float64(p.Minor),
				Y: float64(p.Major),
			})
		}
	}
	return polyline
}

type PointJoiner struct {
	minor         int
	bucketSize    int
	buckets       [][]JoinerLine
	lines         []JoinerLine
	maxMajorDelta float64
}

func NewPointJoiner(bucketSize, maxMajor int, maxMajorDelta float64) *PointJoiner {
	numBuckets := maxMajor / bucketSize
	if maxMajor%bucketSize > 0 {
		numBuckets++
	}
	return &PointJoiner{
		bucketSize:    bucketSize,
		buckets:       make([][]JoinerLine, numBuckets),
		maxMajorDelta: maxMajorDelta,
	}
}

func IsLineAdmissable(line JoinerLine) bool {
	// it's not a line if it doesn't have enough points.
	if len(line) < cfg.VectorizeMinLinePixelLength {
		return false
	}

	totalWidth := 0.0
	for _, p := range line {
		totalWidth += float64(p.Width)
	}
	avgWidth := totalWidth / float64(len(line))
	return len(line) > int(avgWidth*1.3)
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
			if int(lastPoint.Minor) < pj.minor-1 {
				// Add the line to the output lines slice if it passes filtering criteria.
				if IsLineAdmissable(line) {
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
			if math.Abs(float64(major-lastPoint.Major)) <= pj.maxMajorDelta {
				point := JoinerLinePoint{Major: major, Minor: float32(pj.minor), Width: width}
				pj.buckets[bucketIdx][i] = append(line, point)
				return
			}
		}
	}

	// If the point can't be added to any of the existing lines, create a new line in the bucket
	pj.buckets[pointBucketIdx] = append(pj.buckets[pointBucketIdx],
		JoinerLine{{Major: major, Minor: float32(pj.minor), Width: width}})
}

func (pj *PointJoiner) JoinerLines() []JoinerLine {
	// Advance Y twice to flush out all buckets
	pj.NextMinor()
	pj.NextMinor()

	return pj.lines

	// Note: should prevent any further calls to NextMinor() or AddPoint()
}
