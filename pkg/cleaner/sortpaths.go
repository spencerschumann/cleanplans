package cleaner

import (
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"fmt"
	"os"
)

func SortPaths(svg *SVGXMLNode) {
	// Sort paths to minimize travel distance
	// For now, assume each child is a single independent path with no sub-children.

	minX, minY, maxX, maxY := svg.Bounds()
	tree := newPathTree(minX, minY, maxX, maxY)
	sorted := []*svgpath.SubPath{}

	pathStyles := map[*svgpath.SubPath]map[string]string{}

	for _, child := range svg.Children {
		for _, path := range child.Path {
			tree.addPath(path)
			pathStyles[path] = child.style
			//fmt.Fprintln(os.Stderr, "added one path to tree")
		}
	}

	x, y := 0.0, svg.HeightInMM()
	for {
		nearestList := tree.findNearest(x, y, 1)
		fmt.Fprintf(os.Stderr, "findNearest %g %g returned %d\n", x, y, len(nearestList))
		if len(nearestList) == 0 {
			//fmt.Fprintln(os.Stderr, "findNearest returned 0 results")
			break
		}
		nearest := nearestList[0]
		tree.removePath(nearest)

		// TODO: reverse the path if the end is nearest
		if distance(x, y, nearest, false) < distance(x, y, nearest, true) {
			styles := pathStyles[nearest]
			nearest = nearest.Reverse()
			pathStyles[nearest] = styles
		}
		x, y = nearest.EndPoint()
		sorted = append(sorted, nearest)
	}
	fmt.Fprintf(os.Stderr, "Total number of paths: %d\n", len(sorted))

	svg.Children = nil
	for _, path := range sorted {
		svg.Children = append(svg.Children, &SVGXMLNode{
			XMLName: xml.Name{
				Space: "http://www.w3.org/2000/svg",
				Local: "path",
			},
			// TODO: need to keep track of categories within the tree...for now just let them all collapse into black
			Category: CategoryFullCut,
			Path:     []*svgpath.SubPath{path},
			style:    pathStyles[path],
		})
	}
}
