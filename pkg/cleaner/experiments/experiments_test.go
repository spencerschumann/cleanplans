package experiments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindHorizontalLines(t *testing.T) {
	tests := []struct {
		blob     Blob
		expected []LineSegment
	}{
		{
			blob: Blob{Runs: []Run{
				{X1: 0, X2: 10, Y: 1},
			}},
			expected: []LineSegment{
				{
					A:     Point{X: 0, Y: 1.5},
					B:     Point{X: 10, Y: 1.5},
					Width: 1,
				},
			},
		},
		{
			blob: Blob{Runs: []Run{
				{X1: 0, X2: 10, Y: 1},
				{X1: 0, X2: 10, Y: 2},
			}},
			expected: []LineSegment{
				{
					A:     Point{X: 0, Y: 2},
					B:     Point{X: 10, Y: 2},
					Width: 2,
				},
			},
		},
		{
			blob: Blob{Runs: []Run{
				{X1: 0, X2: 10, Y: 1},
				{X1: 0, X2: 10, Y: 2},
				{X1: 0, X2: 1, Y: 3},
				{X1: 0, X2: 1, Y: 4},
				{X1: 0, X2: 1, Y: 5},
				{X1: 0, X2: 1, Y: 6},
				{X1: 0, X2: 1, Y: 7},
			}},
			expected: []LineSegment{
				{
					A:     Point{X: 0, Y: 2},
					B:     Point{X: 10, Y: 2},
					Width: 2,
				},
			},
		},
	}

	for _, test := range tests {
		actual := FindHorizontalLines(test.blob)
		if diff := cmp.Diff(test.expected, actual); diff != "" {
			t.Errorf("incorrect result for %v: %s", test.blob, diff)
		}
	}
}

/*func naiveSumXX(run Run) float64 {
	sum := 0.0
	for x := run.X1 + 0.5; x < run.X2; x++ {
		sum += x * x
	}
	return sum
}

func naiveSumXXX(run Run) float64 {
	sum := 0.0
	for x := run.X1 + 0.5; x < run.X2; x++ {
		sum += x * x * x
	}
	return sum
}

func FuzzComputeRunSums(f *testing.F) {
	f.Add(0, 0)
	f.Add(1, 2)
	f.Add(1, 3)
	f.Add(-1, 1)
	f.Add(10, 100)
	f.Add(-9, -2)
	f.Fuzz(func(t *testing.T, x1, x2 int) {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		run := Run{X1: float64(x1), X2: float64(x2)}
		sumXX := computeRunSumXX(run)
		expectedXX := naiveSumXX(run)
		if sumXX != expectedXX {
			t.Errorf("Run %v: expected sumXX to be %f, but got %f", run, expectedXX, sumXX)
		}

		sumXXX := computeRunSumXXX(run)
		expectedXXX := naiveSumXXX(run)
		if sumXXX != expectedXXX {
			t.Errorf("Run %v: expected sumXXX to be %f, but got %f", run, expectedXXX, sumXXX)
		}
	})
}*/
