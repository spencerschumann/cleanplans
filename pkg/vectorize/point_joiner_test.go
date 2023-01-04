package vectorize_test

import (
	"cleanplans/pkg/vectorize"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPointJoiner(t *testing.T) {
	tests := []struct {
		Name   string
		Input  []vectorize.LinePoint
		Output []vectorize.Line
	}{
		{
			Name: "single point",
			Input: []vectorize.LinePoint{
				{X: 1, Y: 1},
			},
			Output: nil,
		},
		{
			Name: "two vertical lines",
			Input: []vectorize.LinePoint{
				{X: 1, Y: 1}, {X: 5, Y: 1},
				{X: 1, Y: 2}, {X: 5, Y: 2},
			},
			Output: []vectorize.Line{
				{{X: 1, Y: 1}, {X: 1, Y: 2}},
				{{X: 5, Y: 1}, {X: 5, Y: 2}},
			},
		},

		{
			Name: "two 45 degree diagonal lines",
			Input: []vectorize.LinePoint{
				{X: 1, Y: 1}, {X: 10, Y: 1},
				{X: 2, Y: 2}, {X: 9, Y: 2},
			},
			Output: []vectorize.Line{
				{{X: 1, Y: 1}, {X: 2, Y: 2}},
				{{X: 10, Y: 1}, {X: 9, Y: 2}},
			},
		},

		{
			Name: "one nearly vertical line and one diagonal slightly over 45 degrees",
			Input: []vectorize.LinePoint{
				{X: 1, Y: 1}, {X: 10, Y: 1},
				{X: 1.1, Y: 2}, {X: 8.9, Y: 2},
			},
			Output: []vectorize.Line{
				{{X: 1, Y: 1}, {X: 1.1, Y: 2}},
			},
		},
	}

	for _, test := range tests {
		pj := vectorize.PointJoiner{}
		for _, point := range test.Input {
			pj.AddPoint(point)
		}

		lines := pj.Lines()
		sort.Slice(lines, func(i, j int) bool {
			a, b := lines[i][0], lines[j][0]
			if a.Y == b.Y {
				return a.X < b.X
			}
			return a.Y < b.Y
		})

		diff := cmp.Diff(test.Output, lines)
		if diff != "" {
			t.Errorf("test %s: incorrect output: %s", test.Name, diff)
		}
	}
}
