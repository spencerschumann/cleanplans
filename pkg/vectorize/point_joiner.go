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
	minor          int
	bucketSize     int
	buckets        []map[JoinerLinePoint]JoinerLine
	lines          []JoinerLine
	maxMajorDelta  float64
	MinAspectRatio float64
}

func NewPointJoiner(bucketSize, maxMajor int, maxMajorDelta float64) *PointJoiner {
	numBuckets := maxMajor / bucketSize
	if maxMajor%bucketSize > 0 {
		numBuckets++
	}
	buckets := make([]map[JoinerLinePoint]JoinerLine, numBuckets)
	for i := 0; i < numBuckets; i++ {
		buckets[i] = map[JoinerLinePoint]JoinerLine{}
	}
	return &PointJoiner{
		bucketSize:     bucketSize,
		buckets:        buckets,
		maxMajorDelta:  maxMajorDelta,
		MinAspectRatio: 4.0,
	}
}

func (pj *PointJoiner) IsLineAdmissable(line JoinerLine) bool {
	// it's not a line if it doesn't have enough points.
	if len(line) < cfg.VectorizeMinLinePixelLength {
		return false
	}

	totalWidth := 0.0
	for _, p := range line {
		totalWidth += float64(p.Width)
	}
	avgWidth := totalWidth / float64(len(line))
	return len(line) > int(avgWidth*pj.MinAspectRatio)
}

func (pj *PointJoiner) NextMinor() {
	pj.minor++

	// Clear out old lines that have ended, and move them to the lines slice.
	// Otherwise the buckets will continue grow beyond their expected small size
	// and we won't get the desired O(1) performance.
	for _, bucket := range pj.buckets {
		for lineKey, line := range bucket {
			lastPoint := line[len(line)-1]
			// TODO: maybe relax this - rather than ending the line if there's any glitch,
			// it might be better to see if it picks back up within a few more minor values.
			if int(lastPoint.Minor) < pj.minor-1 {
				// Add the line to the output lines slice if it passes filtering criteria.
				if pj.IsLineAdmissable(line) {
					//fmt.Println("Output line:", line)
					pj.lines = append(pj.lines, line)
				}
				delete(bucket, lineKey)
			}
		}
	}
}

func (pj *PointJoiner) overlap(major float32, width int, lastPoint JoinerLinePoint) bool {
	if pj.maxMajorDelta == 0 {
		return major == lastPoint.Major && width == lastPoint.Width
	}

	if math.Abs(float64(major-lastPoint.Major)) > pj.maxMajorDelta {
		return false
	}

	return math.Abs(float64(major-lastPoint.Major)) <= float64(width+lastPoint.Width)/2
}

func (pj *PointJoiner) AddRun(major float32, width int) {
	//fmt.Printf("Run: major=%04.1f minor=%02d width=%02d", major, pj.minor, width)
	//checkedOverlap := false
	// Find the appropriate bucket for the point
	pointBucketIdx := int(major / float32(pj.bucketSize))

	// Check the point's bucket, and the buckets adjacent to it.
	firstBucketIdx := pointBucketIdx - 1
	if firstBucketIdx < 0 {
		firstBucketIdx = 0
	}
	lastBucketIdx := pointBucketIdx + 1
	if lastBucketIdx >= len(pj.buckets) {
		lastBucketIdx = len(pj.buckets) - 1
	}
	for bucketIdx := firstBucketIdx; bucketIdx <= lastBucketIdx; bucketIdx++ {
		// Check if the point can be added to any of the existing lines in the bucket
		for lineKey, line := range pj.buckets[bucketIdx] {
			lastPoint := line[len(line)-1]
			if pj.overlap(major, width, lastPoint) {
				point := JoinerLinePoint{Major: major, Minor: float32(pj.minor), Width: width}
				line = append(line, point)
				if bucketIdx == pointBucketIdx {
					pj.buckets[bucketIdx][lineKey] = line
				} else {
					// Slanted lines move from bucket to bucket along their lengths.
					delete(pj.buckets[bucketIdx], lineKey)
					pj.buckets[pointBucketIdx][lineKey] = line
				}
				return
			}
		}
	}
	/*if !checkedOverlap {
		fmt.Println()
	}*/

	// If the point can't be added to any of the existing lines, create a new line in the bucket
	point := JoinerLinePoint{Major: major, Minor: float32(pj.minor), Width: width}
	pj.buckets[pointBucketIdx][point] = JoinerLine{point}
}

func (pj *PointJoiner) JoinerLines() []JoinerLine {
	// Advance Y twice to flush out all buckets
	pj.NextMinor()
	pj.NextMinor()

	return pj.lines

	// Note: should prevent any further calls to NextMinor() or AddPoint()
}
