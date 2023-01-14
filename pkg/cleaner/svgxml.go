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

	// Attributes for circle elements. This is getting rediculous...really need to break this down.
	CX     float64 `xml:"cx,attr,omitempty"`
	CY     float64 `xml:"cy,attr,omitempty"`
	Radius float64 `xml:"r,attr,omitempty"`

	// inkscape-specific data; inkscape crashes if an extension doesn't preserve the namedview element.
	Docname   string     `xml:"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd docname,attr,omitempty"`
	NamedView *NamedView `xml:"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd namedview,omitempty"`

	Category Category           `xml:"-"`
	Path     []*svgpath.SubPath `xml:"-"`

	style          map[string]string
	styleNameOrder map[string]int
	widthInMM      float64
	heightInMM     float64
	baseUnits      string
}

type NamedView struct {
	ID               string `xml:"id,attr,omitempty"`
	PageColor        string `xml:"pagecolor,attr,omitempty"`
	BorderColor      string `xml:"bordercolor,attr,omitempty"`
	BorderOpacity    string `xml:"borderopacity,attr,omitempty"`
	ObjectTolerance  string `xml:"objecttolerance,attr,omitempty"`
	GridTolerance    string `xml:"gridtolerance,attr,omitempty"`
	ShowGrid         string `xml:"showgrid,attr,omitempty"`
	GuideTolerance   string `xml:"guidetolerance,attr,omitempty"`
	PageShadow       string `xml:"http://www.inkscape.org/namespaces/inkscape pageshadow,attr,omitempty"`
	PageOpacity      string `xml:"http://www.inkscape.org/namespaces/inkscape pageopacity,attr,omitempty"`
	PageCheckerBoard string `xml:"http://www.inkscape.org/namespaces/inkscape pagecheckerboard,attr,omitempty"`
	DocumentUnits    string `xml:"http://www.inkscape.org/namespaces/inkscape document-units,attr,omitempty"`
	Zoom             string `xml:"http://www.inkscape.org/namespaces/inkscape zoom,attr,omitempty"`
	CX               string `xml:"http://www.inkscape.org/namespaces/inkscape cx,attr,omitempty"`
	CY               string `xml:"http://www.inkscape.org/namespaces/inkscape cy,attr,omitempty"`
	WindowWidth      string `xml:"http://www.inkscape.org/namespaces/inkscape window-width,attr,omitempty"`
	WindowHeight     string `xml:"http://www.inkscape.org/namespaces/inkscape window-height,attr,omitempty"`
	WindowX          string `xml:"http://www.inkscape.org/namespaces/inkscape window-x,attr,omitempty"`
	WindowY          string `xml:"http://www.inkscape.org/namespaces/inkscape window-y,attr,omitempty"`
	WindowMaximized  string `xml:"http://www.inkscape.org/namespaces/inkscape window-maximized,attr,omitempty"`
	CurrentLayer     string `xml:"http://www.inkscape.org/namespaces/inkscape current-layer,attr,omitempty"`
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
		for _, path := range node.Path {
			// TODO: this just tracks start/end of each segment; this is correct for lines
			// but not for curves.
			minX = math.Min(minX, path.X)
			maxX = math.Max(maxX, path.X)
			minY = math.Min(minY, path.Y)
			maxY = math.Max(maxY, path.Y)
			for _, d := range path.DrawTo {
				minX = math.Min(minX, d.X)
				maxX = math.Max(maxX, d.X)
				minY = math.Min(minY, d.Y)
				maxY = math.Max(maxY, d.Y)
			}
		}
	}
	return
}

func (n *SVGXMLNode) RemoveEmptyPaths() {
	nodeIndex := 0
	for _, node := range n.Children {
		pathIndex := 0
		for _, path := range node.Path {
			if len(path.DrawTo) > 0 {
				node.Path[pathIndex] = path
				pathIndex++
			}
		}
		node.Path = node.Path[:pathIndex]
		if pathIndex > 0 {
			n.Children[nodeIndex] = node
			nodeIndex++
		}
	}
	n.Children = n.Children[:nodeIndex]
}

func (n *SVGXMLNode) Marshal() ([]byte, error) {
	// Reserialize attributes
	if n.widthInMM != 0 && n.heightInMM != 0 {
		n.Width = FormatNumber(n.widthInMM) + "mm"
		n.Height = FormatNumber(n.heightInMM) + "mm"
	}
	for _, child := range n.Children {
		// Back to a path string
		if child.Path != nil {
			child.D = svgpath.ToString(child.Path)
		}

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
