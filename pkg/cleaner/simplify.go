package cleaner

import (
	"cleanplans/pkg/svgpath"
)

// Simplify simplfies the paths in the node, joining adjacent
// paths and removing redundant points.
func Simplify(svg *SVGXMLNode) {
	// Group children by category
	groups := map[Category][]*SVGXMLNode{}
	for _, node := range svg.Children {
		groups[node.Category] = append(groups[node.Category], node)
	}

	var tree *pathTree

	tryMerge := func(path *svgpath.SubPath, start bool) bool {
		var x, y float64
		if start {
			x, y = path.StartPoint()
		} else {
			x, y = path.EndPoint()
		}

		// IMPORTANT: this needs to be configurable. It's being changed here from the assumption
		// of mm units to pixel units, as part of the debugging. In the end all of these operations
		// should operate on standardized, real-world mm units, because the end goal is to translate
		// the input into something the needle cutter can process, and the limiting factor there
		// is the needle width (around 0.8 mm) and the accuracy of the machine itself.
		//const maxDist = 0.01
		const maxDist = 20.0

		neighbors := tree.findNeighbors(path, x, y, 0, maxDist)
		// For now, only merge if there's just one neighbor with the same start/end point.
		// Could look at n-way intersections in the future, but for now just deal with the
		// common case of adjacent pairs of paths.
		if len(neighbors) != 1 {
			return false
		}

		other := neighbors[0]
		ds := distance(x, y, other, true)
		de := distance(x, y, other, false)
		merged := false
		if de < ds {
			merged = mergePaths(path, start, other, false, 360)
		} else {
			merged = mergePaths(path, start, other, true, 360)
		}
		if !merged {
			return false
		}
		// Remove path; will add back afterwards with updated coords
		tree.removePath(path)
		tree.removePath(other)
		// add the path back with its updated endpoints
		tree.addPath(path)

		return true
	}

	for _, category := range []Category{CategoryFullCut, CategoryScore, CategoryPaperCut, CategoryOptional, CategoryCrease} {
		minX, minY, maxX, maxY := svg.Bounds()
		tree = newPathTree(minX-1, minY-1, maxX+1, maxY+1)

		for _, node := range groups[category] {
			for _, path := range node.Path {
				tree.addPath(path)
			}
		}

		for _, node := range groups[category] {
			for _, path := range node.Path {
				// Skip paths that have already been merged,
				// then repeatedly try merging until this path can't merge further.
				for len(path.DrawTo) > 0 &&
					(tryMerge(path, true) || tryMerge(path, false)) {
				}
			}
		}
	}

	// Filter out paths that were merged away
	svg.RemoveEmptyPaths()

	for _, node := range svg.Children {
		for _, path := range node.Path {
			path.Simplify()
		}
	}
}
