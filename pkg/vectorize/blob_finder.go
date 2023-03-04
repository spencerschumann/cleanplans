package vectorize

import (
	"cleanplans/pkg/geometry"
	"math"

	//"math"
	"sort"

	"github.com/chewxy/math32"
)

// Run is a horizontal set of adjacent, same-colored pixels.
type Run struct {
	X1       Float
	X2       Float
	Y        Float
	Eclipsed bool
}

type Runs []*Run

type Connection struct {
	A        *Blob
	B        *Blob
	Location geometry.Point
	// TODO: optional extra point for weakly connected blobs (i.e., dashed or broken lines)
}

// Blob is a sequence of adjacent runs. Runs are adjacent
// if the Y values differ by exactly 1, and the X values differ by at most 1.
type Blob struct {
	Runs        Runs
	Connections map[*Connection]struct{}
	Transposed  bool
}

// AverageWidth returns the average width of the runs.
func (runs Runs) AverageWidth() Float {
	var wSum Float = 0.0
	for _, run := range runs {
		wSum += run.X2 - run.X1
	}
	count := Float(len(runs))
	return wSum / count
}

func (blob *Blob) BestFitArc() geometry.Arc {
	circle := blob.BestFitCircle()
	if circle.Radius == 0 {
		return geometry.Arc{}
	}

	// Find start and end points; use the mid point of the first and last run, and find the intersection
	// between the circle and the lines between the center point and each of these end points.

	runToPoint := func(run *Run) geometry.Point {
		p := geometry.Point{
			X: (run.X1 + run.X2) / 2,
			Y: run.Y + 0.5,
		}
		if blob.Transposed {
			p.X, p.Y = p.Y, p.X
		}
		// Find where this point's ray from the center intersects the circle
		d := p.Distance(circle.Center)
		p = p.Minus(circle.Center).Scale(circle.Radius / d)
		result := p.Add(circle.Center)
		//fmt.Printf("Point: %#v, d: %f, radius: %f, result: %#v, result.Diistance(circle.Center): %f\n",
		//	p, d, circle.Radius, result, result.Distance(circle.Center))
		return result
	}

	// To find the arc direction, look at the cross product between the segment from center to start and the
	// segment from center to the middle run of the blob.
	start := runToPoint(blob.Runs[0])
	mid := runToPoint(blob.Runs[len(blob.Runs)/2])
	end := runToPoint(blob.Runs[len(blob.Runs)-1])

	clockwise := (start.Minus(circle.Center)).CrossProductZ(mid.Minus(circle.Center)) > 0

	//fmt.Printf("Circle: %#v\n  start: %#v\n  mid: %#v\n  end: %#v\n  clockwise: %t\n", circle, start, mid, end, clockwise)

	return geometry.Arc{
		Start:     start,
		End:       end,
		Center:    circle.Center,
		Clockwise: clockwise,
	}
}

// Use the technique from https://dtcenter.org/sites/default/files/community-code/met/docs/write-ups/circle_fit.pdf
func (blob *Blob) BestFitCircle() geometry.Circle {
	// Calculate centroid (average x and y coordinates) of pixels in the blob
	// Note: this was already calculated in ToPolyline; if that function is always called first,
	// the result could be used to avoid recomputing it here. Might be worth converting blob
	// to a struct and memoizing the result.
	var n, sumX, sumY Float
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
	var Suv, Suu, Svv, Suuu, Svvv, Suvv, Svuu Float
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
	//radius := math32.Sqrt(uc*uc + vc*vc + (Suu+Svv)/n)
	radius := Float(math.Sqrt(float64(uc*uc + vc*vc + (Suu+Svv)/n)))
	xc := uc + avgX
	yc := vc + avgY

	if blob.Transposed {
		xc, yc = yc, xc
	}

	return geometry.Circle{
		Center: geometry.Point{X: xc, Y: yc},
		Radius: radius,
	}
}

func LineFitAcceptable(runs Runs, line geometry.LineSegment) bool {
	halfWidth := runs.AverageWidth() / 2

	//fmt.Printf("Line: %v-%v, width/2: %f\n", p1, p2, width)

	// adjust width (horizontal run length) based on slope of line
	// This may be useful, but for now I'll just use the width as-is.
	/*
		width /= 2
		ray := p2.Minus(p1)
		width = math.Abs(geometry.Point{X: width}.CrossProductZ(ray)) / ray.Magnitude()
	*/

	sumExtra := func(a, b Float) Float {
		if b <= a {
			return 0
		}
		// integrate function y=x-w, from x=a to x=b
		extra := math32.Abs((b*b/2 - halfWidth*b) - (a*a/2 - halfWidth*a))
		//fmt.Printf("    extra %f from x=%f to x=%f\n", extra, a, b)
		return extra / 5
	}

	sumMissing := func(a, b Float) Float {
		missing := math32.Max(0, b-a)
		// if missing > 0 {
		// 	fmt.Printf("    missing %f from x=%f to x=%f\n", missing, a, b)
		// }
		return missing / 3
	}

	// Line equation: ax + by + c = 0
	a := line.A.Y - line.B.Y
	b := line.B.X - line.A.X
	c := line.A.X*line.B.Y - line.B.X*line.A.Y

	var error, xError, xAvgError Float
	for _, run := range runs {
		y := run.Y + 0.5
		// substitute y into line equation and solve for x:
		//   x = (-by - c) / a
		x := (-b*y - c) / a

		// How much of the run lies within width of x? Expected x from e1 to e2, run x adjusted to center on x, from r1 to r2.
		e1 := -halfWidth
		e2 := halfWidth
		r1 := run.X1 - x
		r2 := run.X2 - x
		mid := (run.X1 + run.X2) / 2
		xAvgError += x - mid
		xError += math32.Abs(x - mid)
		//fmt.Printf("  run %v, x=%f, x-delta=%f, e1=%f, e2=%f, r1=%f, r2=%f\n", run, x, math.Abs(x-mid), e1, e2, r1, r2)
		if math32.Abs(mid-x) < .5 && math32.Abs(r1-e1) < 1 && math32.Abs(r2-e2) < 1 {
			// consider these runs to be equivalent - no error
			// TODO: make threshold configurable?
			// TODO: also track overall midpoint drift? For example, a perfect vertical line that's shifted 0.49 pixels to the right should show an error.
			//fmt.Println("   No error")
		} else {
			error += sumMissing(e1, math32.Min(e2, r1))
			error += sumMissing(math32.Max(e1, r2), e2)
			error += sumExtra(r1, math32.Min(r2, e1))
			error += sumExtra(math32.Max(r1, e2), r2)
		}
	}

	//fmt.Printf("  total error: %f, xError/count: %f, xAvgError/count: %f\n", error, xError/count, xAvgError/count)

	// TODO: configurable threshold, and should it depend on width and/or length of line?
	//return error < 1.0+0.02*math.Sqrt(count) && math.Abs(xAvgError) < .01*math.Sqrt(count) && xError < 0.2*math.Sqrt(count)
	return error < 1.0 && math32.Abs(xAvgError) < 0.3 //&& xError <
}

func ToLineSegment(runs Runs) geometry.LineSegment {
	// use linear regression to find the best-fit line to the blob
	var n, Sx, Sxx, Sy, Syy, Sxy float64
	//minX := math.Inf(+1)
	//maxX := math.Inf(-1)
	minY := math.Inf(+1)
	maxY := math.Inf(-1)
	for _, run := range runs {
		y := float64(run.Y) + 0.5
		width := float64(run.X2 - run.X1)
		n += width
		Sy += width * y
		runSumX := width * float64(run.X1+run.X2) / 2
		Sx += runSumX
		Sxy += runSumX * y

		//a := run.X1 + 0.5
		//b := run.X2 - 0.5
		//Sxx += (2*a*a + 2*a*b - a + 2*b*b + b) * (a - b - 1) / -6

		// TODO: I can't remember why this Sxx definition is different than the Sxx definition in BestFitCircle.
		// I would have thought the BestFitCircle definition would work, but it doesn't.
		Sxx += width*float64(run.X1)*float64(run.X1) + float64(run.X1)*width*width + width*width*width/3

		Syy += y * y * width
		//minX = math.Min(minX, float64(run.X1))
		//maxX = math.Max(maxX, float64(run.X2))
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}

	//betaDenominatorX := n*Sxx - Sx*Sx
	betaDenominatorY := n*Syy - Sy*Sy
	/*if betaDenominatorY == 0 {
		fmt.Println("*** betaDenominatorY == 0:", n, Syy, Sy)
	}*/
	var p1, p2 geometry.Point
	/*if false && betaDenominatorY < betaDenominatorX {
		// mostly horizontal line
		betaNumerator := n*Sxy - Sx*Sy
		beta := betaNumerator / betaDenominatorX
		alpha := Sy/n - beta*Sx/n
		p1 = geometry.Point{X: minX, Y: alpha + beta*minX}
		p2 = geometry.Point{X: maxX, Y: alpha + beta*maxX}
	} else*/{
		// mostly vertical line
		betaNumerator := n*Sxy - Sx*Sy
		beta := betaNumerator / betaDenominatorY
		alpha := Sx/n - beta*Sy/n
		p1 = geometry.Point{Y: Float(minY), X: Float(alpha + beta*minY)}
		p2 = geometry.Point{Y: Float(maxY), X: Float(alpha + beta*maxY)}

		/*if math32.IsNaN(p1.X) {
			fmt.Println("*** X is NaN:", alpha, beta, minY, betaNumerator, betaDenominatorY, Sx, Sy, n)
		}*/
	}
	return geometry.LineSegment{A: p1, B: p2}
}

func findSplit(runs Runs) int {
	// First choice: try to find a corner directly.
	min := math32.Inf(+1)
	max := math32.Inf(-1)
	var firstMin, lastMin, firstMax, lastMax int
	minRunning := false
	maxRunning := false
	for i, run := range runs {
		if run.X1 < min {
			min = run.X1
			firstMin = i
			lastMin = i
			minRunning = true
		} else if minRunning && run.X1 == min {
			lastMin = i
		} else {
			minRunning = false
		}
		if max < run.X2 {
			max = run.X2
			firstMax = i
			lastMax = i
			maxRunning = true
		} else if maxRunning && max == run.X2 {
			lastMax = i
		} else {
			maxRunning = false
		}
		//fmt.Printf(">run=%+v, min=%f max=%f firstMin=%d lastMin=%d firstMax=%d lastMax=%d\n",
		//	run, min, max, firstMin, lastMin, firstMax, lastMax)
	}
	width := int(runs.AverageWidth())
	//fmt.Printf("len(runs)=%d width=%d min=%f max=%f firstMin=%d lastMin=%d firstMax=%d lastMax=%d\n",
	//	len(runs), width, min, max, firstMin, lastMin, firstMax, lastMax)
	// If there's a clump of max or min runs that's not too large and that's
	// not too close to the ends, use the middle of the clump as the split point.
	maxRun := int(math32.Max(Float(width*10), Float(len(runs)/4)))
	if width < firstMin && lastMin < len(runs)-width && (lastMin-firstMin) < maxRun {
		return (firstMin+lastMin)/2 + 1
	}
	if width < firstMax && lastMax < len(runs)-width && (lastMax-firstMax) < maxRun {
		return (firstMax+lastMax)/2 + 1
	}

	// Binary search isn't ideal, because it searches for a line that's just _barely_,
	// acceptable, but it does find a line that's acceptable.
	return sort.Search(len(runs), func(i int) bool {
		runs := runs[i:]
		line := ToLineSegment(runs)
		return LineFitAcceptable(runs, line)
	})
}

func splitify(runs Runs) []geometry.LineSegment {
	if len(runs) < 2 {
		return nil
	}

	line := ToLineSegment(runs)
	// Is this a good fit? If so, return this line.
	if LineFitAcceptable(runs, line) {
		return []geometry.LineSegment{line}
	}

	split := findSplit(runs)
	if split < 2 || len(runs)-2 < split {
		// no good split found - use the remainder regardless of fit quality
		line := ToLineSegment(runs)
		//fmt.Printf("Fallback to remainder, runs=%+v, line=%v\n", remaining, line)
		return []geometry.LineSegment{line}
	}

	// recurse for each half, instead of taking this one as good directly.
	return append(
		splitify(runs[:split]),
		splitify(runs[split:])...,
	)
}

func intersection(a, b geometry.LineSegment) geometry.Point {
	// see https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection
	x1, y1 := a.A.X, a.A.Y
	x2, y2 := a.B.X, a.B.Y
	x3, y3 := b.A.X, b.A.Y
	x4, y4 := b.B.X, b.B.Y
	pXNum := (x1*y2-y1*x2)*(x3-x4) - (x1-x2)*(x3*y4-y3*x4)
	pYNum := (x1*y2-y1*x2)*(y3-y4) - (y1-y2)*(x3*y4-y3*x4)
	denominator := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	return geometry.Point{
		X: pXNum / denominator,
		Y: pYNum / denominator,
	}
}

func conjoin(segs []geometry.LineSegment) geometry.Polyline {
	if len(segs) == 0 {
		return nil
	}
	polyline := geometry.Polyline{segs[0].A}
	for i := 0; i < len(segs)-1; i++ {
		prev := segs[i]
		next := segs[i+1]
		isect := intersection(prev, next)
		dpn := math32.Max(2, prev.B.Distance(next.A))
		dpbi := prev.B.Distance(isect)
		dpai := prev.A.Distance(isect)
		dnai := next.A.Distance(isect)
		dnbi := next.B.Distance(isect)
		if dnai < dpn*10 && dpbi < dpn*10 && dpbi < dpai && dnai < dnbi {
			/*if math32.IsNaN(isect.X) || math32.IsNaN(isect.Y) {
				fmt.Println("Bad isect for", prev, next, ":", isect)
				panic("bad isect")
			}*/
			polyline = append(polyline, isect)
		} else {
			// The intersection either doesn't exist or is too far away.
			// TODO: be more sophisticated. For now just average the points.
			delta := next.A.Minus(prev.B)
			ratio := next.Length() / (prev.Length() + next.Length())
			p := prev.B.Add(delta.Scale(ratio))
			/*if math32.IsNaN(p.X) || math32.IsNaN(p.Y) {
				fmt.Println("Bad averaging for", prev, next, ":", p)
				panic("bad isect")
			}*/
			polyline = append(polyline, p)
		}
	}
	polyline = append(polyline, segs[len(segs)-1].B)
	return polyline
}

func (blob *Blob) ToPolyline() (geometry.Polyline, []geometry.LineSegment) {
	segs := splitify(blob.Runs[:]) // TODO: do I need this slice copy here?
	if blob.Transposed {
		for i := range segs {
			seg := &segs[i]
			seg.A.X, seg.A.Y = seg.A.Y, seg.A.X
			seg.B.X, seg.B.Y = seg.B.Y, seg.B.X
		}
	}
	polyline := conjoin(segs)

	/*for _, point := range polyline {
		// TODO: make math32/math and Float all part of a float package, to handle the float32/float64 split
		if math32.IsNaN(point.X) || math32.IsNaN(point.Y) {
			fmt.Println("Bad blob!!!!")
			for _, run := range blob.Runs {
				fmt.Println("  Run:", *run)
			}
			for _, seg := range segs {
				fmt.Println("  Seg:", seg)
			}
			break
		}
	}*/

	return polyline, segs
}

type BlobFinder struct {
	y           int
	bucketSize  int
	numBuckets  int
	buckets     []map[*Run]*Blob
	prevBuckets []map[*Run]*Blob
	blobs       map[*Blob]struct{}
	connections map[*Connection]struct{}
	//Runs        []Runs
	//TrackRuns   bool

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
		//Runs:        []Runs{{}},
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
	/*for i := 0; i < bf.numBuckets; i++ {
		bf.buckets[i] = map[*Run]*Blob{}
	}*/
}

func (bf *BlobFinder) NextY() {
	bf.y++
	bf.makeBuckets()
	/*if bf.TrackRuns {
		bf.Runs = append(bf.Runs, Runs{})
	}*/
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

func (bf *BlobFinder) AddRun(run *Run) {
	var runBlob *Blob
	/*if bf.TrackRuns {
		bf.Runs[bf.y] = append(bf.Runs[bf.y], run)
	}*/
	firstBucketIdx, lastBucketIdx := bf.runBuckets(run)
	connected := map[*Blob]bool{}
	for i := firstBucketIdx; i <= lastBucketIdx; i++ {
		// Check if the run can be added to any of the existing blobs in the bucket
		for prevRun, blob := range bf.prevBuckets[i] {
			if !run.overlap(prevRun) {
				continue
			}
			if runBlob == nil {
				if blob.Runs[len(blob.Runs)-1].Y == Float(bf.y)-1 {
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
				x1 := math32.Max(prevRun.X1, run.X1)
				x2 := math32.Min(prevRun.X2, run.X2)
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
		if bf.buckets[i] == nil {
			bf.buckets[i] = map[*Run]*Blob{}
		}
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

	width := func(run *Run) Float {
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
		if math32.Abs(lastW-w) > 3 && math32.Max(lastW, w) > math32.Min(lastW, w)*2 {
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

	bf.splitBlobs()

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
