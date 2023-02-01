package experiments

// Run is a horizontal set of adjacent, same-colored pixels.
type Run struct {
	X1 float64
	X2 float64
	Y  float64
}

// Blob is a sequence of adjacent runs. Runs are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type Blob []Run

func computeRunSumXX(run Run) float64 {
	a := run.X1 + 0.5
	b := run.X2 - 0.5
	return (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6
}

func computeRunSumXXX(run Run) float64 {
	a := run.X1 + 0.5
	b := run.X2 - 0.5
	return (a*a - a + b*b + b) * (a + b) * (a - b - 1) / -4
}
