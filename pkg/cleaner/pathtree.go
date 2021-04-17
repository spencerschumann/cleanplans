package cleaner

import (
	"cleanplans/pkg/svgpath"
	"math"
	"sort"

	"github.com/asim/quadtree"
)

var zeroPoint = quadtree.NewPoint(0, 0, nil)

type pathTree struct {
	quadTree *quadtree.QuadTree
}

func newPathTree(minX, minY, maxX, maxY float64) *pathTree {
	midX := (maxX + minX) / 2
	midY := (maxY + minY) / 2
	aabb := quadtree.NewAABB(
		quadtree.NewPoint(midX, midY, nil),
		quadtree.NewPoint(maxX-midX, maxY-midY, nil))
	return &pathTree{
		quadTree: quadtree.New(aabb, 0, nil),
	}
}

func (t *pathTree) addPath(path *svgpath.SubPath) {
	if len(path.DrawTo) == 0 {
		return
	}

	addOne := func(x, y float64) {
		point := quadtree.NewPoint(x, y, nil)
		points := t.quadTree.KNearest(quadtree.NewAABB(point, zeroPoint), 1, nil)
		if len(points) > 0 {
			pointX, pointY := points[0].Coordinates()
			if pointX == x && pointY == y {
				// Add the path to the existing list
				paths := points[0].Data().(map[*svgpath.SubPath]struct{})
				paths[path] = struct{}{}
				return
			}
		}
		paths := map[*svgpath.SubPath]struct{}{path: {}}
		t.quadTree.Insert(quadtree.NewPoint(x, y, paths))
	}

	lastDrawTo := path.DrawTo[len(path.DrawTo)-1]
	addOne(path.X, path.Y)
	addOne(lastDrawTo.X, lastDrawTo.Y)

}

func (t *pathTree) removePath(path *svgpath.SubPath) {
	removeOne := func(x, y float64) {
		point := quadtree.NewPoint(x, y, nil)
		points := t.quadTree.KNearest(quadtree.NewAABB(point, zeroPoint), 1, nil)
		if len(points) > 0 {
			pointX, pointY := points[0].Coordinates()
			if pointX == x && pointY == y {
				paths := points[0].Data().(map[*svgpath.SubPath]struct{})
				delete(paths, path)
			}
		}
	}
	removeOne(path.X, path.Y)
	ex, ey := path.EndPoint()
	removeOne(ex, ey)
}

func (t *pathTree) findNeighbors(path *svgpath.SubPath, x, y, minDist, maxDist float64) []*svgpath.SubPath {
	var neighbors []*svgpath.SubPath
	nearAABB := quadtree.NewAABB(
		quadtree.NewPoint(x, y, nil),
		quadtree.NewPoint(maxDist, maxDist, nil),
	)
	points := t.quadTree.Search(nearAABB)
	for _, point := range points {
		otherPaths := point.Data().(map[*svgpath.SubPath]struct{})
		if minDist > 0 {
			// Skip all paths that share exact points with this one
			if _, found := otherPaths[path]; found {
				continue
			}
		}
		for other := range otherPaths {
			if other != path {
				d := math.Min(distance(x, y, other, true), distance(x, y, other, false))
				if minDist <= d && d <= maxDist {
					neighbors = append(neighbors, other)
				}
			}
		}
	}

	// Sort by distance from the given point
	sort.Slice(neighbors, func(i, j int) bool {
		di := math.Min(distance(x, y, neighbors[i], true), distance(x, y, neighbors[i], false))
		dj := math.Min(distance(x, y, neighbors[j], true), distance(x, y, neighbors[j], false))
		return di < dj
	})
	return neighbors
}
