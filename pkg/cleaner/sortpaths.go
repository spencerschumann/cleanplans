package cleaner

import (
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"math"
)

func SortPaths(svg *SVGXMLNode) {
	// Sort paths to minimize travel distance
	// For now, assume each child is a single independent path with no sub-children.

	minX, minY, maxX, maxY := svg.Bounds()
	tree := newPathTree(minX-1, minY-1, maxX+1, maxY+1)
	sorted := []*svgpath.SubPath{}

	pathNodes := map[*svgpath.SubPath]*SVGXMLNode{}

	for _, child := range svg.Children {
		for _, path := range child.Path {
			tree.addPath(path)
			pathNodes[path] = child
		}
	}

	// Start from point within bounding box nearest 0, 0
	x := math.Max(0.0, minX)
	y := math.Min(svg.HeightInMM(), maxY) // machine's 0, 0 = svg's 0, height
	for {
		nearestList := tree.findNearest(x, y, 1)
		//fmt.Fprintf(os.Stderr, "findNearest %g %g returned %d\n", x, y, len(nearestList))
		if len(nearestList) == 0 {
			//fmt.Fprintln(os.Stderr, "findNearest returned 0 results")
			break
		}
		nearest := nearestList[0]
		tree.removePath(nearest)

		// reverse the path if the end is nearest
		if distance(x, y, nearest, false) < distance(x, y, nearest, true) {
			node := pathNodes[nearest]
			nearest = nearest.Reverse()
			pathNodes[nearest] = node
		}
		x, y = nearest.EndPoint()
		sorted = append(sorted, nearest)
	}
	//fmt.Fprintf(os.Stderr, "Total number of paths: %d\n", len(sorted))

	svg.Children = nil
	for _, path := range sorted {
		node := pathNodes[path]
		//fmt.Fprintf(os.Stderr, "Post-sort object: %s\n", node.ID)
		svg.Children = append(svg.Children, &SVGXMLNode{
			XMLName: xml.Name{
				Space: "http://www.w3.org/2000/svg",
				Local: "path",
			},
			Category: node.Category,
			Path:     []*svgpath.SubPath{path},
			style:    node.style,
			ID:       node.ID,
		})
	}
}
