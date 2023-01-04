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
	lines []Line
}

func (pj *PointJoiner) AddPoint(point LinePoint) {
	// Create a new line if no lines exist
	if len(pj.lines) == 0 {
		pj.lines = append(pj.lines, Line{point})
		return
	}

	// Check if the point can be added to any of the existing lines
	for i, line := range pj.lines {
		lastPoint := line[len(line)-1]
		if point.Y == lastPoint.Y+1 && math.Abs(float64(point.X-lastPoint.X)) <= 1 {
			pj.lines[i] = append(line, point)
			return
		}
	}

	// If the point can't be added to any of the existing lines, create a new line
	pj.lines = append(pj.lines, Line{point})
}

func (pj *PointJoiner) Lines() []Line {
	var lines []Line
	for _, line := range pj.lines {
		// Each Line must have at least 2 points.
		if len(line) >= 2 {
			lines = append(lines, line)
		}
	}
	return lines
}
