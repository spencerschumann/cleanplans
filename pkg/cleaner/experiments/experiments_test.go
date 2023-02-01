package experiments

import "testing"

func naiveSumXX(run Run) float64 {
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
}
