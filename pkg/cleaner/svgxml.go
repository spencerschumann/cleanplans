package cleaner

import (
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"math"
	"strconv"
)

type SVGXMLNode struct {
	XMLName   xml.Name
	Width     string        `xml:"width,attr,omitempty"`
	Height    string        `xml:"height,attr,omitempty"`
	ViewBox   string        `xml:"viewBox,attr,omitempty"`
	Version   string        `xml:"version,attr,omitempty"`
	ID        string        `xml:"id,attr,omitempty"`
	Styles    string        `xml:"style,attr,omitempty"`
	D         string        `xml:"d,attr,omitempty"`
	Transform string        `xml:"transform,attr,omitempty"`
	Children  []*SVGXMLNode `xml:",any"`

	// inkscape-specific data; inkscape crashes if an extension doesn't preserve the namedview element.
	Docname string `xml:"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd docname,attr,omitempty"`
	NamedView *NamedView `xml:"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd namedview,omitempty"`

	style          map[string]string
	styleNameOrder map[string]int
	category       Category
	path           []*svgpath.SubPath
	widthInMM      float64
	heightInMM     float64
	baseUnits      string
}

type NamedView struct {
	ID string `xml:"id,attr,omitempty"`
	PageColor string `xml:"pagecolor,attr,omitempty"`
	BorderColor string `xml:"bordercolor,attr,omitempty"`
	BorderOpacity string `xml:"borderopacity,attr,omitempty"`
	ObjectTolerance string `xml:"objecttolerance,attr,omitempty"`
	GridTolerance string `xml:"gridtolerance,attr,omitempty"`
	ShowGrid string `xml:"showgrid,attr,omitempty"`
	GuideTolerance string `xml:"guidetolerance,attr,omitempty"`
	PageShadow string `xml:"http://www.inkscape.org/namespaces/inkscape pageshadow,attr,omitempty"`
	PageOpacity string `xml:"http://www.inkscape.org/namespaces/inkscape pageopacity,attr,omitempty"`
	PageCheckerBoard string `xml:"http://www.inkscape.org/namespaces/inkscape pagecheckerboard,attr,omitempty"`
	DocumentUnits string `xml:"http://www.inkscape.org/namespaces/inkscape document-units,attr,omitempty"`
	Zoom string `xml:"http://www.inkscape.org/namespaces/inkscape zoom,attr,omitempty"`
	CX string `xml:"http://www.inkscape.org/namespaces/inkscape cx,attr,omitempty"`
	CY string `xml:"http://www.inkscape.org/namespaces/inkscape cy,attr,omitempty"`
	WindowWidth string `xml:"http://www.inkscape.org/namespaces/inkscape window-width,attr,omitempty"`
	WindowHeight string `xml:"http://www.inkscape.org/namespaces/inkscape window-height,attr,omitempty"`
	WindowX string `xml:"http://www.inkscape.org/namespaces/inkscape window-x,attr,omitempty"`
	WindowY string `xml:"http://www.inkscape.org/namespaces/inkscape window-y,attr,omitempty"`
	WindowMaximized string `xml:"http://www.inkscape.org/namespaces/inkscape window-maximized,attr,omitempty"`
	CurrentLayer string `xml:"http://www.inkscape.org/namespaces/inkscape current-layer,attr,omitempty"`
}

func Parse(data []byte) (*SVGXMLNode, error) {
	var svg SVGXMLNode
	err := xml.Unmarshal(data, &svg)
	return &svg, err
}

func (n *SVGXMLNode) Bounds() (minX, minY, maxX, maxY float64) {
	minX = math.Inf(1)
	maxX = math.Inf(-1)
	minY = math.Inf(1)
	maxY = math.Inf(-1)
	for _, node := range n.Children {
		for _, path := range node.path {
			// TODO: this just tracks start/end; this is correct for lines
			// but not for curves.
			lastX, lastY := path.EndPoint()
			minX = math.Min(minX, math.Min(path.X, lastX))
			maxX = math.Max(maxX, math.Max(path.X, lastX))
			minY = math.Min(minY, math.Min(path.Y, lastY))
			maxY = math.Max(maxY, math.Max(path.Y, lastY))
		}
	}
	return
}

func (n *SVGXMLNode) RemoveEmptyPaths() {
	nodeIndex := 0
	for _, node := range n.Children {
		pathIndex := 0
		for _, path := range node.path {
			if len(path.DrawTo) > 0 {
				node.path[pathIndex] = path
				pathIndex++
			}
		}
		node.path = node.path[:pathIndex]
		if pathIndex > 0 {
			n.Children[nodeIndex] = node
			nodeIndex++
		}
	}
	n.Children = n.Children[:nodeIndex]
}

func (n *SVGXMLNode) Marshal() ([]byte, error) {
	// Reserialize attributes
	n.Width = FormatNumber(n.widthInMM) + "mm"
	n.Height = FormatNumber(n.heightInMM) + "mm"
	for _, child := range n.Children {
		// Back to a path string
		child.D = svgpath.ToString(child.path)

		// Reserialize style to capture changes
		child.serializeStyle()

		// SVG namespace at root is enough
		child.XMLName.Space = ""
	}

	return xml.MarshalIndent(n, "", "  ")
}

// TODO: where do these two belong???
func ParseNumber(n string) float64 {
	val, _ := strconv.ParseFloat(n, 64)
	return val
}

func FormatNumber(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}
