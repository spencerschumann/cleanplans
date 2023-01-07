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
		Input  [][]float32
		Output []vectorize.JoinerLine
	}{
		{
			Name:   "single point",
			Input:  [][]float32{{1}},
			Output: nil,
		},
		{
			Name: "two vertical lines",
			Input: [][]float32{
				{1, 5},
				{1, 5},
			},
			Output: []vectorize.JoinerLine{
				{{Major: 1, Minor: 0}, {Major: 1, Minor: 1}},
				{{Major: 5, Minor: 0}, {Major: 5, Minor: 1}},
			},
		},

		{
			Name: "two 45 degree diagonal lines",
			Input: [][]float32{
				{1, 10},
				{2, 9},
			},
			Output: []vectorize.JoinerLine{
				{{Major: 1, Minor: 0}, {Major: 2, Minor: 1}},
				{{Major: 10, Minor: 0}, {Major: 9, Minor: 1}},
			},
		},

		{
			Name: "one nearly vertical line and one diagonal slightly over 45 degrees",
			Input: [][]float32{
				{1, 10},
				{1.1, 8.9},
			},
			Output: []vectorize.JoinerLine{
				{{Major: 1, Minor: 0}, {Major: 1.1, Minor: 1}},
			},
		},

		// TODO: add tests for behavior around bucket boundaries
	}

	for _, test := range tests {
		pj := vectorize.NewPointJoiner(10, 19)
		for _, row := range test.Input {
			for _, major := range row {
				pj.AddRun(major, 0)
			}
			pj.NextMinor()
		}

		lines := pj.JoinerLines()
		sort.Slice(lines, func(i, j int) bool {
			a, b := lines[i][0], lines[j][0]
			if a.Minor == b.Minor {
				return a.Major < b.Major
			}
			return a.Minor < b.Minor
		})

		diff := cmp.Diff(test.Output, lines)
		if diff != "" {
			t.Errorf("test %s: incorrect output: %s", test.Name, diff)
		}
	}
}
