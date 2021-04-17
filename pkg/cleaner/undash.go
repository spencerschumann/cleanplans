package cleaner

import (
	"cleanplans/pkg/svgpath"
	"math"
)

func distance(x, y float64, path *svgpath.SubPath, fromStart bool) float64 {
	var dx, dy float64
	if fromStart {
		dx, dy = path.StartPoint()
	} else {
		dx, dy = path.EndPoint()
	}
	dx, dy = dx-x, dy-y
	return math.Sqrt(dx*dx + dy*dy)
}

func mergePaths(path *svgpath.SubPath, pathStart bool, join *svgpath.SubPath, joinStart bool, limitAngle float64) bool {
	// glue adds a line from the end of a to the start of b, and stores the result in path.
	glue := func(a, b *svgpath.SubPath) bool {
		connectWithLine := false

		// Check angle and distance before gluing
		{
			// Get last segment of a
			aex, aey := a.EndPoint()
			asx, asy := a.StartPoint()
			if len(a.DrawTo) > 1 {
				penultimate := a.DrawTo[len(a.DrawTo)-2]
				asx, asy = penultimate.X, penultimate.Y
			}

			// Get first segment of b
			bsx, bsy := b.StartPoint()
			bex, bey := b.EndPoint()
			if len(a.DrawTo) > 1 {
				first := b.DrawTo[0]
				bex, bey = first.X, first.Y
			}

			dax, day := aex-asx, aey-asy
			dbx, dby := bex-bsx, bey-bsy

			dot := dax*dbx + day*dby
			distA := math.Sqrt(dax*dax + day*day)
			distB := math.Sqrt(dbx*dbx + dby*dby)

			angle := math.Acos(dot/distA/distB) * 180 / math.Pi

			if angle > limitAngle {
				return false
			}

			dx := aex - bsx
			dy := aey - bsy
			distPaths := math.Sqrt(dx*dx + dy*dy)
			if distPaths > 0.1 { // TODO: configurable
				connectWithLine = true
			}
		}

		drawTo := a.DrawTo
		if connectWithLine {
			drawTo = append(a.DrawTo,
				&svgpath.DrawTo{Command: svgpath.LineTo, X: b.X, Y: b.Y})
		}
		drawTo = append(drawTo, b.DrawTo...)
		/// Careful, `path` can be aliased to either `a` or `b`.
		// Don't update `path` too soon and thereby modify a or b before reading them.
		path.X = a.X
		path.Y = a.Y
		path.DrawTo = drawTo
		return true
	}

	var a, b *svgpath.SubPath
	if pathStart == false && joinStart == true {
		a, b = path, join
	} else if joinStart == false && pathStart == true {
		a, b = join, path
	} else if pathStart == false && joinStart == false {
		a, b = path, join.Reverse()
	} else if pathStart == true && joinStart == true {
		a, b = path.Reverse(), join
	}

	if !glue(a, b) {
		return false
	}

	// Clear out the joined path
	join.DrawTo = nil

	return true
}

// Undash consolidates dashed lines into solid lines.
func Undash(svg *SVGXMLNode) {
	// Group children by category
	groups := map[Category][]*SVGXMLNode{}
	for _, node := range svg.Children {
		groups[node.category] = append(groups[node.category], node)
	}

	var tree *pathTree
	var mergeCounts map[*svgpath.SubPath]int

	mergeOne := func(path *svgpath.SubPath, pathStart bool, join *svgpath.SubPath, joinStart bool) bool {
		merged := mergePaths(path, pathStart, join, joinStart, 45) // TODO: configurable limit angle
		if !merged {
			return false
		}
		// Remove path; will add back afterwards with updated coords
		tree.removePath(path)
		tree.removePath(join)
		// add the path back with its updated endpoints
		tree.addPath(path)

		mergeCounts[path] += 1 + mergeCounts[join]
		return true
	}

	// TODO: configurable max/min distance
	maxDist := 4.0
	minDist := 0.5

	tryMerge := func(path *svgpath.SubPath, start bool) bool {
		var x, y float64
		if start {
			x, y = path.StartPoint()
		} else {
			x, y = path.EndPoint()
		}

		neighbors := tree.findNeighbors(path, x, y, minDist, maxDist)
		for _, neighbor := range neighbors {
			ds := distance(x, y, neighbor, true)
			de := distance(x, y, neighbor, false)
			if ds < de {
				// TODO: also check for similar slopes at start/end points?
				// For now, just join segments if they are within the right distance range.
				// TODO: configurable min/max distance values
				if minDist <= ds && ds <= maxDist && mergeOne(path, start, neighbor, true) {
					return true
				}
			} else {
				if minDist <= de && de <= maxDist && mergeOne(path, start, neighbor, false) {
					return true
				}
			}
		}
		return false
	}

	for _, category := range []Category{CategoryScore, CategoryPaperCut, CategoryOptional, CategoryCrease} {
		minX, minY, maxX, maxY := svg.Bounds()
		tree = newPathTree(minX, minY, maxX, maxY)
		mergeCounts = map[*svgpath.SubPath]int{}

		for _, node := range groups[category] {
			for _, path := range node.path {
				tree.addPath(path)
			}
		}

		for _, node := range groups[category] {
			for _, path := range node.path {
				// Skip paths that have already been merged
				if len(path.DrawTo) == 0 {
					continue
				}

				// TODO: too many layers, too much complexity. Collapse this down to a merge wrapper
				// that will try a merge, without regard to sorting by distance
				if tryMerge(path, true) {
					continue
				}

				tryMerge(path, false)
			}
		}
	}

	// Filter out paths that were merged away
	svg.RemoveEmptyPaths()

	// Check for possible loops, and join the ends
	for _, node := range svg.Children {
		for _, path := range node.path {
			// Require at least 4 dashes for a path that forms a loop
			if mergeCounts[path] >= 3 && len(path.DrawTo) >= 4 {
				sx, sy := path.StartPoint()
				ex, ey := path.EndPoint()
				dx, dy := sx-ex, sy-ey
				dist := math.Sqrt(dx*dx + dy*dy)
				if minDist <= dist && dist <= maxDist {
					path.DrawTo = append(path.DrawTo, &svgpath.DrawTo{
						Command: svgpath.ClosePath,
						X:       path.X,
						Y:       path.Y,
					})
				}
			}
		}
	}
}
