package vectorize

import (
	"cleanplans/pkg/geometry"
	"sort"

	"github.com/asim/quadtree"
)

type LineSet struct {
	quadTree *quadtree.QuadTree
	width    float64
	height   float64
}

func NewLineSet(width, height float64) *LineSet {
	aabb := quadtree.NewAABB(
		quadtree.NewPoint(width/2, height/2, nil),
		quadtree.NewPoint(width/2+10, height/2+10, nil),
	)
	return &LineSet{
		quadTree: quadtree.New(aabb, 0, nil),
		width:    width,
		height:   height,
	}
}

var zeroPoint = quadtree.NewPoint(0, 0, nil)

// AddLine adds line to the LineSet.
func (ls *LineSet) AddLine(line geometry.Polyline) {
	if len(line) < 2 {
		return
	}

	addOne := func(x, y float64) {
		point := quadtree.NewPoint(x, y, nil)
		points := ls.quadTree.KNearest(quadtree.NewAABB(point, zeroPoint), 1, nil)
		if len(points) > 0 {
			pointX, pointY := points[0].Coordinates()
			if pointX == x && pointY == y {
				// Add the path to the existing list
				lines := points[0].Data().(map[*geometry.Polyline]struct{})
				lines[&line] = struct{}{}
				return
			}
		}
		lines := map[*geometry.Polyline]struct{}{&line: {}}
		ls.quadTree.Insert(quadtree.NewPoint(x, y, lines))
	}

	lastPoint := line[len(line)-1]
	addOne(line[0].X, line[0].Y)
	addOne(lastPoint.X, lastPoint.Y)
}

/*func (t *pathTree) removePath(path *svgpath.SubPath) {
	removeOne := func(x, y float64) {
		point := quadtree.NewPoint(x, y, nil)
		points := t.quadTree.KNearest(quadtree.NewAABB(point, zeroPoint), 1, nil)
		if len(points) > 0 {
			pointX, pointY := points[0].Coordinates()
			if pointX == x && pointY == y {
				paths := points[0].Data().(map[*svgpath.SubPath]struct{})
				delete(paths, path)
				if len(paths) == 0 {
					t.quadTree.Remove(points[0])
				}
			}
		}
	}
	removeOne(path.X, path.Y)
	ex, ey := path.EndPoint()
	removeOne(ex, ey)
}*/

func (ls *LineSet) FindNeighbors(p geometry.Point, maxDist float64) []geometry.Polyline {
	var neighbors []geometry.Polyline
	nearAABB := quadtree.NewAABB(
		quadtree.NewPoint(p.X, p.Y, nil),
		quadtree.NewPoint(maxDist, maxDist, nil),
	)
	points := ls.quadTree.Search(nearAABB)
	for _, point := range points {
		otherLines := point.Data().(map[*geometry.Polyline]struct{})
		for other := range otherLines {
			d := other.EndpointDistance(p)
			if d <= maxDist {
				neighbors = append(neighbors, *other)
			}
		}
	}

	// Sort by distance from the given point
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].EndpointDistance(p) < neighbors[j].EndpointDistance(p)
	})
	return neighbors
}

/*func (ls *LineSet) findNearest(p geometry.Point) []*svgpath.SubPath {
	aabb := quadtree.NewAABB(
		quadtree.NewPoint(p.X, p.Y, nil),
		quadtree.NewPoint(ls.width, ls.height, nil),
	)
	// TODO: I don't think KNearest works like it should. Probably need to file a bug report.
	points := ls.quadTree.KNearest(aabb, 5, nil)

	var nearest []*geometry.Polyline
	for _, point := range points {
		lines := point.Data().(map[*geometry.Polyline]struct{})
		for line := range lines {
			nearest = append(nearest, line)
		}
	}

	sort.Slice(nearest, func(i, j int) bool {
		return nearest[i].EndpointDistance(p) < nearest[j].EndpointDistance(p)
	})

	if len(nearest) > maxCount {
		nearest = nearest[:maxCount]
	}

	return nearest
}*/