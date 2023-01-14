//go:build js && wasm

package main

import (
	"bytes"
	"cleanplans/pkg/cleaner"
	"cleanplans/pkg/geometry"
	"cleanplans/pkg/svgpath"
	"cleanplans/pkg/vectorize"
	"encoding/xml"
	"fmt"
	"image/png"
	"math"
	"runtime"
	"strconv"
	"syscall/js"
)

func main() {
	funcs := []struct {
		name string
		fn   func(js.Value, []js.Value) any
	}{
		{"goCleanPlans", goCleanPlans},
		{"goTestSimplifyPolyline", goTestSimplifyPolyline},
	}
	for _, fn := range funcs {
		js.Global().Set(fn.name, js.FuncOf(fn.fn))
	}

	<-make(chan any, 0)
}

type bounds struct {
	initialized bool
	scale       float64
	xOffset     float64
	yOffset     float64
	width       float64
	height      float64
	allPoints   []geometry.Point

	// Super hack alert! Throwing this in here for easy access
	lastMsg string
}

func (b *bounds) initialize(step geometry.Step) {
	if b.initialized {
		return
	}
	xMin := math.Inf(1)
	xMax := math.Inf(-1)
	yMin := math.Inf(1)
	yMax := math.Inf(-1)
	for _, p := range step.Points {
		xMin = math.Min(p.X, xMin)
		xMax = math.Max(p.X, xMax)
		yMin = math.Min(p.Y, yMin)
		yMax = math.Max(p.Y, yMax)
	}
	width := xMax - xMin
	height := yMax - yMin
	if width > height {
		b.scale = 800 / width
	} else {
		b.scale = 600 / height
	}
	margin := 100.0

	b.width = width*b.scale + margin*2
	b.height = height*b.scale + margin*2
	b.xOffset = margin - xMin*b.scale
	b.yOffset = margin - yMin*b.scale

	for _, p := range step.Points {
		b.allPoints = append(b.allPoints, p)
	}
}

func (b *bounds) BaseSVG(step geometry.Step) cleaner.SVGXMLNode {
	pathNode := cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "path"},
		Styles:  "fill:none;stroke:#770000;stroke-width:1;stroke-linecap:butt;stroke-linejoin:miter;stroke-miterlimit:4;stroke-opacity:1",
	}

	svg := cleaner.SVGXMLNode{
		XMLName:  xml.Name{Local: "svg"},
		Width:    strconv.FormatFloat(b.width, 'g', 5, 64),
		Height:   strconv.FormatFloat(b.height, 'g', 5, 64),
		Children: []*cleaner.SVGXMLNode{&pathNode},
	}

	stepPoints := map[geometry.Point]struct{}{}
	for _, p := range step.Points {
		stepPoints[p] = struct{}{}
	}

	addPoint := func(x, y float64, point geometry.Point) {
		styles := "fill:#cccccc"
		if _, ok := stepPoints[point]; ok {
			styles = "fill:#33dd33"
		}
		svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
			XMLName: xml.Name{Local: "circle"},
			CX:      x,
			CY:      y,
			Radius:  b.width / 100,
			Styles:  styles,
		})
	}

	path := svgpath.SubPath{
		X: b.transformX(step.Points[0].X),
		Y: b.transformY(step.Points[0].Y),
	}
	addPoint(path.X, path.Y, step.Points[0])
	for _, p := range b.allPoints {
		drawTo := svgpath.DrawTo{
			Command: svgpath.LineTo,
			X:       b.transformX(p.X),
			Y:       b.transformY(p.Y),
		}
		path.DrawTo = append(path.DrawTo, &drawTo)
		addPoint(drawTo.X, drawTo.Y, p)
	}
	pathNode.Path = []*svgpath.SubPath{&path}

	return svg
}

func (b *bounds) transformX(x float64) float64 {
	return x*b.scale + b.xOffset
}

func (b *bounds) transformY(y float64) float64 {
	return b.height - (y*b.scale + b.yOffset)
}

// takes step as input, outputs an svg string
func svgSimplifyStep(step geometry.Step, bounds *bounds) cleaner.SVGXMLNode {
	if len(step.Points) == 0 {
		return cleaner.SVGXMLNode{}
	}
	svg := bounds.BaseSVG(step)
	return svg
}

func svgChordStep(step geometry.Step, bounds *bounds) cleaner.SVGXMLNode {
	svg := bounds.BaseSVG(step)
	svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "circle"},
		CX:      bounds.transformX(step.Chord.A.X),
		CY:      bounds.transformY(step.Chord.A.Y),
		Radius:  bounds.width / 80,
		Styles:  "fill:#333333;stroke-width:2;stroke:#ff0000",
	})
	svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "circle"},
		CX:      bounds.transformX(step.Chord.B.X),
		CY:      bounds.transformY(step.Chord.B.Y),
		Radius:  bounds.width / 80,
		Styles:  "fill:#333333;stroke-width:2;stroke:#ff0000",
	})
	pathNode := cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "path"},
		Styles:  "fill:none;stroke:#ff0000;stroke-width:3;stroke-linecap:butt;stroke-linejoin:miter;stroke-miterlimit:4;stroke-opacity:1",
		Path: []*svgpath.SubPath{
			{
				X: bounds.transformX(step.Chord.A.X),
				Y: bounds.transformY(step.Chord.A.Y),
				DrawTo: []*svgpath.DrawTo{
					{
						Command: svgpath.LineTo,
						X:       bounds.transformX(step.Chord.B.X),
						Y:       bounds.transformY(step.Chord.B.Y),
					},
				},
			},
		},
	}
	svg.Children = append(svg.Children, &pathNode)
	return svg
}

func svgMaxDistStep(step geometry.Step, bounds *bounds) cleaner.SVGXMLNode {
	svg := svgChordStep(step, bounds)
	svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "circle"},
		CX:      bounds.transformX(step.FarPoint.X),
		CY:      bounds.transformY(step.FarPoint.Y),
		Radius:  bounds.width / 80,
		Styles:  "fill:#333333;stroke-width:2;stroke:#ff0000",
	})
	return svg
}

func arcToPath(arc geometry.Arc, bounds *bounds) string {
	// Arc path format:
	// A rx ry x-axis-rotation large-arc-flag sweep-flag x y

	radius := arc.Start.Distance(arc.Center) * bounds.scale

	sweepFlag := 0
	if arc.Clockwise {
		sweepFlag = 1
	}

	largeFlag := 0
	crossProduct := arc.Start.Minus(arc.Center).CrossProductZ(arc.End.Minus(arc.Center))
	if (crossProduct > 1) == arc.Clockwise {
		largeFlag = 1
	}

	// using super hack here! TODO: get rid of this hack.
	bounds.lastMsg = fmt.Sprintf("cp=%f, clockwise=%t, sweep=%d, large=%d", crossProduct, arc.Clockwise, sweepFlag, largeFlag)

	return fmt.Sprintf(" M %f %f A %f %f 0 %d %d %f %f ",
		bounds.transformX(arc.Start.X), bounds.transformY(arc.Start.Y),
		radius, radius, largeFlag, sweepFlag,
		bounds.transformX(arc.End.X), bounds.transformY(arc.End.Y),
	)
}

func svgFindArcStep(step geometry.Step, bounds *bounds) cleaner.SVGXMLNode {
	svg := bounds.BaseSVG(step)
	svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "circle"},
		CX:      bounds.transformX(step.Arc.Start.X),
		CY:      bounds.transformY(step.Arc.Start.Y),
		Radius:  bounds.width / 80,
		Styles:  "fill:#33cc33;stroke-width:2;stroke:#ff0000",
	})
	svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "circle"},
		CX:      bounds.transformX(step.Arc.End.X),
		CY:      bounds.transformY(step.Arc.End.Y),
		Radius:  bounds.width / 80,
		Styles:  "fill:#cc3333;stroke-width:2;stroke:#ff0000",
	})
	svg.Children = append(svg.Children, &cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "circle"},
		CX:      bounds.transformX(step.Arc.Center.X),
		CY:      bounds.transformY(step.Arc.Center.Y),
		Radius:  bounds.width / 80,
		Styles:  "fill:#333333;stroke-width:2;stroke:#ff0000",
	})
	pathNode := cleaner.SVGXMLNode{
		XMLName: xml.Name{Local: "path"},
		Styles:  "fill:none;stroke:#ff0000;stroke-width:3;stroke-linecap:butt;stroke-linejoin:miter;stroke-miterlimit:4;stroke-opacity:1",
		D:       arcToPath(step.Arc, bounds),
	}
	svg.Children = append(svg.Children, &pathNode)
	return svg
}

func goTestSimplifyPolyline(this js.Value, args []js.Value) any {
	points := geometry.Polyline{
		{0, -4},
		{0, -3},
		{0, -2},
		{0, -1},
		{0, 0},
		{0.4, 1.5},
		{1.5, 2.6},
		{3.0, 3.0},
		{4.5, 2.6},
		{5.6, 1.5},
		{6, 0},
	}

	steps := make(chan geometry.Step)
	go func() {
		points.Simplify(.1, steps)
		close(steps)
	}()

	var svgSteps []any
	var bounds bounds
	for step := range steps {
		var svg cleaner.SVGXMLNode
		switch step.Name {
		// simplify, chord, result, maxDist, findArc, arcDistanceExceeded,
		case "simplify":
			if len(svgSteps) == 0 {
				bounds.initialize(step)
			}
			svg = svgSimplifyStep(step, &bounds)
		case "chord":
			continue
			svg = svgChordStep(step, &bounds)
		case "maxDist":
			svg = svgMaxDistStep(step, &bounds)
		case "findArc":
			svg = svgFindArcStep(step, &bounds)
		}
		data, _ := svg.Marshal()
		svgSteps = append(svgSteps, map[string]any{
			"data": string(data),
			"msg":  bounds.lastMsg})
		bounds.lastMsg = ""
	}

	return svgSteps
}

// goCleanPlans is the main entry point to cleanplans from JavaScript.
func goCleanPlans(this js.Value, args []js.Value) any {
	image := args[0]
	imageLen := image.Length()
	width := args[1].Int()
	height := args[2].Int()
	bitsPerPixel := args[3].Int()
	fmt.Printf("Go CleanPlans called: %d bytes, %d, %d, %d\n", imageLen, width, height, bitsPerPixel)

	data := make([]byte, imageLen)
	js.CopyBytesToGo(data, image)

	ci := vectorize.PDFJSImageToColorImage(data, width, height, bitsPerPixel)

	// drop the data array now and run a GC cycle to conserve memory
	data = nil
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Println("Stats (sys/mallocs):", stats.Sys, stats.Mallocs)

	// Let's do a histogram of ci for debug.
	hist := make([]int, 9)
	for _, c := range ci.Data {
		hist[c] += 1
	}
	for k, v := range hist {
		fmt.Printf("Color histogram slice: %d %d\n", k, v)
	}

	svg := vectorize.Vectorize(ci)

	var buf bytes.Buffer
	err := png.Encode(&buf, ci)
	if err != nil {
		fmt.Printf("Error generating png image: %s\n", err)
		return nil
	}

	fmt.Printf("Encoded png, %d bytes\n", buf.Len())

	u8Array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(u8Array, buf.Bytes())

	return map[string]any{
		"png": u8Array,
		"svg": svg,
	}
}
