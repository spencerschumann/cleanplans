package vectorize

import (
	"cleanplans/pkg/geometry"
	"math"
)

// Run is a horizontal set of adjacent, same-colored pixels.
type Run struct {
	X1       float64
	X2       float64
	Y        float64
	Eclipsed bool
}

type Connection struct {
	A        *Blob
	B        *Blob
	Location geometry.Point
	// TODO: optional extra point for weakly connected blobs (i.e., dashed or broken lines)
}

// Blob is a sequence of adjacent runs. Runs are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type Blob struct {
	Runs        []*Run
	Connections map[*Connection]struct{}
	Transposed  bool
}

// Use the technique from https://dtcenter.org/sites/default/files/community-code/met/docs/write-ups/circle_fit.pdf
func (blob *Blob) BestFitCircle() geometry.Circle {
	// Calculate centroid (average x and y coordinates) of pixels in the blob
	// Note: this was already calculated in ToPolyline; if that function is always called first,
	// the result could be used to avoid recomputing it here. Might be worth converting blob
	// to a struct and memoizing the result.
	n := 0.0
	sumX := 0.0
	sumY := 0.0
	for _, run := range blob.Runs {
		y := run.Y + 0.5
		width := run.X2 - run.X1
		n += width
		sumY += width * y
		runSumX := width * (run.X1 + run.X2) / 2
		sumX += runSumX
	}
	avgX := sumX / n
	avgY := sumY / n

	// Calculate Sxxx sums for Eq. 4 and Eq. 5
	Suv := 0.0
	Suu := 0.0
	Svv := 0.0
	Suuu := 0.0
	Svvv := 0.0
	Suvv := 0.0
	Svuu := 0.0
	for _, run := range blob.Runs {
		u1 := run.X1 - avgX
		u2 := run.X2 - avgX
		v := run.Y + 0.5 - avgY
		width := u2 - u1
		runSu := width * (u1 + u2) / 2
		Suv += runSu * v
		a := u1 + 0.5
		b := u2 - 0.5
		runSuu := (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6
		Suu += runSuu
		Svuu += runSuu * v
		runSvv := v * v * width
		Svv += runSvv
		Svvv += runSvv * v
		Suvv += runSu * v * v
		Suuu += (a*a - a + b*b + b) * (a + b) * (a - b - 1) / -4
	}

	// Now solve the system of equations Eq. 4 and Eq. 5, substituting variables to match this format:
	// 	a1 * uc + b1 * vc = c1
	// 	a2 * uc + b1 * vc = c2
	a1 := Suu
	b1 := Suv
	c1 := (Suuu + Suvv) / 2
	a2 := Suv
	b2 := Svv
	c2 := (Svvv + Svuu) / 2
	det := a1*b2 - a2*b1
	if det == 0 {
		// fail - can't find a suitable circle center
		return geometry.Circle{}
	}
	uc := (c1*b2 - c2*b1) / det
	vc := (a1*c2 - a2*c1) / det

	// Substitute uc and uv into Eq. 6 to compute radius
	radius := math.Sqrt(uc*uc + vc*vc + (Suu+Svv)/n)
	xc := uc + avgX
	yc := vc + avgY

	return geometry.Circle{
		Center: geometry.Point{X: xc, Y: yc},
		Radius: radius,
	}
}

func (blob *Blob) ToPolyline() geometry.Polyline {
	// use linear regression to find the best-fit line to the blob
	n := 0.0
	Sx := 0.0
	Sxx := 0.0
	Sy := 0.0
	Syy := 0.0
	Sxy := 0.0
	minX := math.Inf(+1)
	maxX := math.Inf(-1)
	minY := math.Inf(+1)
	maxY := math.Inf(-1)
	for _, run := range blob.Runs {
		y := run.Y + 0.5
		width := run.X2 - run.X1
		n += width
		Sy += width * y
		runSumX := width * (run.X1 + run.X2) / 2
		Sx += runSumX
		Sxy += runSumX * y

		//a := run.X1 + 0.5
		//b := run.X2 - 0.5
		//Sxx += (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6

		// TODO: I can't remember why this Sxx definition is different than the Sxx definition in BestFitCircle.
		// I would have thought the BestFitCircle definition would work, but it doesn't.
		Sxx += width*run.X1*run.X1 + run.X1*width*width + width*width*width/3

		Syy += y * y * width
		minX = math.Min(minX, run.X1)
		maxX = math.Max(maxX, run.X2)
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}

	betaDenominatorX := n*Sxx - Sx*Sx
	betaDenominatorY := n*Syy - Sy*Sy
	if betaDenominatorY < betaDenominatorX {
		// mostly horizontal line
		betaNumerator := n*Sxy - Sx*Sy
		beta := betaNumerator / betaDenominatorX
		alpha := Sy/n - beta*Sx/n
		return geometry.Polyline{
			{X: minX, Y: alpha + beta*minX},
			{X: maxX, Y: alpha + beta*maxX},
		}
	} else {
		// mostly vertical line
		betaNumerator := n*Sxy - Sx*Sy
		beta := betaNumerator / betaDenominatorY
		alpha := Sx/n - beta*Sy/n
		return geometry.Polyline{
			{Y: minY, X: alpha + beta*minY},
			{Y: maxY, X: alpha + beta*maxY},
		}
	}
}

type BlobFinder struct {
	y           int
	bucketSize  int
	numBuckets  int
	buckets     []map[*Run]*Blob
	prevBuckets []map[*Run]*Blob
	blobs       map[*Blob]struct{}
	connections map[*Connection]struct{}
	Runs        [][]*Run
	TrackRuns   bool

	//blobs *quadtree.QuadTree
}

func NewBlobFinder(bucketSize, maxX, maxY int) *BlobFinder {
	numBuckets := maxX / bucketSize
	if maxX%bucketSize > 0 {
		numBuckets++
	}

	bf := &BlobFinder{
		bucketSize:  bucketSize,
		numBuckets:  numBuckets,
		blobs:       map[*Blob]struct{}{},
		connections: map[*Connection]struct{}{},
		Runs:        [][]*Run{{}},
	}

	// First call adds a row of buckets, second call makes another
	// and moves the first row to prevBuckets.
	bf.makeBuckets()
	bf.makeBuckets()

	return bf
}

func (bf *BlobFinder) makeBuckets() {
	bf.prevBuckets = bf.buckets
	bf.buckets = make([]map[*Run]*Blob, bf.numBuckets)
	for i := 0; i < bf.numBuckets; i++ {
		bf.buckets[i] = map[*Run]*Blob{}
	}
}

func (bf *BlobFinder) NextY() {
	bf.y++
	bf.makeBuckets()
	if bf.TrackRuns {
		bf.Runs = append(bf.Runs, []*Run{})
	}
}

// overlap returns true if the two runs overlap, including diagonally
func (r *Run) overlap(other *Run) bool {
	return r.X1 <= other.X2 && other.X1 <= r.X2
}

func (bf *BlobFinder) runBuckets(run *Run) (int, int) {
	// diagonally adjacent runs are treated as connected, so extend the x values by 1 to compensate.
	first := int(run.X1-1) / bf.bucketSize
	last := int(run.X2+1) / bf.bucketSize

	if first < 0 {
		first = 0
	}
	if last >= len(bf.buckets) {
		last = len(bf.buckets) - 1
	}

	return first, last
}

func (bf *BlobFinder) newBlob() *Blob {
	blob := &Blob{}
	bf.blobs[blob] = struct{}{}
	return blob
}

func (bf *BlobFinder) connect(a, b *Blob, location geometry.Point) {
	// TODO: maybe useful to track connected run indexes also
	c := &Connection{
		A:        a,
		B:        b,
		Location: location,
	}
	if a.Connections == nil {
		a.Connections = map[*Connection]struct{}{}
	}
	if b.Connections == nil {
		b.Connections = map[*Connection]struct{}{}
	}
	a.Connections[c] = struct{}{}
	b.Connections[c] = struct{}{}
	bf.connections[c] = struct{}{}
}

func (bf *BlobFinder) AddRun(x1, x2 float64) {
	var runBlob *Blob
	run := &Run{X1: x1, X2: x2, Y: float64(bf.y)}
	if bf.TrackRuns {
		bf.Runs[bf.y] = append(bf.Runs[bf.y], run)
	}
	firstBucketIdx, lastBucketIdx := bf.runBuckets(run)
	connected := map[*Blob]bool{}
	for i := firstBucketIdx; i <= lastBucketIdx; i++ {
		// Check if the run can be added to any of the existing blobs in the bucket
		for prevRun, blob := range bf.prevBuckets[i] {
			if !run.overlap(prevRun) {
				continue
			}
			if runBlob == nil {
				if blob.Runs[len(blob.Runs)-1].Y == float64(bf.y)-1 {
					// add run to this blob
					runBlob = blob
				} else {
					// blob already had a run added on this Y line; start a new blob instead
					runBlob = bf.newBlob()
				}
				runBlob.Runs = append(runBlob.Runs, run)
			}

			if blob != runBlob && !connected[blob] {
				// Run has already been added to a blob; any other overlapping prevRuns are part of connected blobs.
				// add a connection point at the midpoint of the overlap between the runs
				x1 := math.Max(prevRun.X1, run.X1)
				x2 := math.Min(prevRun.X2, run.X2)
				location := geometry.Point{
					X: (x1 + x2) / 2,
					Y: (prevRun.Y+run.Y)/2 + 0.5,
				}
				bf.connect(blob, runBlob, location)
				connected[blob] = true
			}
		}
	}

	// If the run wasn't added to any of the existing blobs, create a new blob
	if runBlob == nil {
		runBlob = bf.newBlob()
		runBlob.Runs = append(runBlob.Runs, run)
	}

	// track this blob in the buckets belonging to this run
	for i := firstBucketIdx; i <= lastBucketIdx; i++ {
		bf.buckets[i][run] = runBlob
	}
}

func split(blob *Blob) []*Blob {
	// might eventually want this, not sure.
	/*lenCounts := map[int]int{}
	for _, run := range blob.Runs {
		width := int(run.X2 - run.X1 + 0.5)
		lenCounts[width]++
	}
	lengths := []int{}
	for l := range lenCounts {
		lengths = append(lengths, l)
	}
	sort.Slice(lengths, func(i, j int) bool {
		return lenCounts[lengths[i]] > lenCounts[lengths[j]]
	})*/

	runs := blob.Runs

	width := func(run *Run) float64 {
		return run.X2 - run.X1
	}

	var blobs []*Blob
	firstIndex := 0
	lastW := width(runs[0])
	for i, run := range runs {
		if i == 0 {
			continue
		}
		w := width(run)
		// If successive runs differ by at least a minimum amount and factor, split there.
		if math.Abs(lastW-w) > 3 && math.Max(lastW, w) > math.Min(lastW, w)*2 {
			newBlob := &Blob{
				Runs: runs[firstIndex:i],
			}
			blobs = append(blobs, newBlob)
			firstIndex = i
			blob.Runs = runs[i:]
		}
		lastW = w
	}

	if len(blobs) > 0 {
		blobs = append(blobs, blob)
	}

	return blobs
}

func (bf *BlobFinder) splitBlobs() {
	for blob := range bf.blobs {
		blobs := split(blob)
		if len(blobs) > 1 {
			// TODO: recast all the connections
			// For the first pass, don't worry about them.
			delete(bf.blobs, blob)
			for _, blob := range blobs {
				bf.blobs[blob] = struct{}{}
			}
		}
	}
}

func (bf *BlobFinder) Blobs() []*Blob {
	// Further calls to NextY or AddRun are undefined.
	bf.buckets = nil

	//bf.splitBlobs()

	// TODO: maybe there was no need to put blobs in a map like this...we'll see.
	var blobs []*Blob
	for blob := range bf.blobs {
		blobs = append(blobs, blob)
	}

	return blobs
}

func (bf *BlobFinder) Connections() []*Connection {
	var connections []*Connection
	for c := range bf.connections {
		connections = append(connections, c)
	}
	return connections
}
