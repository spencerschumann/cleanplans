package vectorize_test

import (
	"cleanplans/pkg/cfg"
	"cleanplans/pkg/vectorize"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPointJoiner(t *testing.T) {
	tests := []struct {
		Name   string
		Input  [][]float32
		Output []vectorize.Blob
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
			Output: []vectorize.Blob{
				{{X: 1, Y: 0}, {X: 1, Y: 1}},
				{{X: 5, Y: 0}, {X: 5, Y: 1}},
			},
		},

		{
			Name: "two 45 degree diagonal lines",
			Input: [][]float32{
				{1, 10},
				{2, 9},
			},
			Output: []vectorize.Blob{
				{{X: 1, Y: 0}, {X: 2, Y: 1}},
				{{X: 10, Y: 0}, {X: 9, Y: 1}},
			},
		},

		{
			Name: "one nearly vertical line and one diagonal slightly over 45 degrees",
			Input: [][]float32{
				{1, 10},
				{1.1, 8.9},
			},
			Output: []vectorize.Blob{
				{{X: 1, Y: 0}, {X: 1.1, Y: 1}},
			},
		},

		// TODO: add tests for behavior around bucket boundaries
	}

	oldMax := cfg.VectorizeMaxRunLength
	defer func() {
		cfg.VectorizeMaxRunLength = oldMax
	}()
	cfg.VectorizeMinLinePixelLength = 2
	for _, test := range tests {
		pj := vectorize.NewBlobFinder(10, 19)
		for _, row := range test.Input {
			for _, major := range row {
				pj.AddRun(major, 0)
			}
			pj.NextY()
		}

		lines := pj.Blobs()
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
