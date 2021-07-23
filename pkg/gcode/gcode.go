package gcode

import (
	"cleanplans/pkg/cleaner"
	"cleanplans/pkg/svgpath"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"os"
)

func Generate(svg *cleaner.SVGXMLNode) {
	{
		cleaner.SortPaths(svg)

		dashedStyle := "fill:none;stroke:#c900ce;stroke-width:0.5;stroke-linecap:butt;stroke-linejoin:miter;stroke-opacity:1;stroke-miterlimit:4;stroke-dasharray:0.5,0.5;stroke-dashoffset:0"

		// Place dashed lines between each segment to show the pen-up "travel" movements
		travel := &cleaner.SVGXMLNode{
			XMLName: xml.Name{
				Space: "http://www.w3.org/2000/svg",
				Local: "path",
			},
			ID:     "travel_indicators",
			Styles: dashedStyle,
		}

		totalTravel := 0.0
		lastX, lastY := 0.0, svg.HeightInMM()
		travelTo := func(x, y float64) {
			dx := lastX - x
			dy := lastY - y
			totalTravel += math.Sqrt(dx*dx + dy*dy)
			travel.Path = append(travel.Path, &svgpath.SubPath{
				X: lastX,
				Y: lastY,
				DrawTo: []*svgpath.DrawTo{
					{
						Command: svgpath.LineTo,
						X:       x,
						Y:       y,
					},
				},
			})
		}

		var iterate func(svg *cleaner.SVGXMLNode)
		iterate = func(svg *cleaner.SVGXMLNode) {
			fmt.Fprintf(os.Stderr, "Entering node with %d children\n", len(svg.Children))
			for _, child := range svg.Children {
				for _, path := range child.Path {
					dist := math.Abs(path.X-lastX) + math.Abs(path.Y-lastY)
					if dist > 0.1 {
						travelTo(path.X, path.Y)
					}
					lastX, lastY = path.EndPoint()
				}
				iterate(child)
			}
		}
		iterate(svg)
		travelTo(0, svg.HeightInMM())
		svg.Children = append(svg.Children, travel)

		fmt.Fprintln(os.Stderr, "Total travel:", totalTravel)

		outXML, err := svg.Marshal()
		if err != nil {
			log.Fatalf("marshal error: %s", err)
		}
		fmt.Println(string(outXML))
		return
	}

	xHome := 0.0
	yHome := 0.0
	zHeight := 0.0
	xyTravelRate := 10000.0
	xyFeedRate := 1500.0
	zFeedRate := 1500.0
	penUpSpeed := 3300.0
	penUpZ := 2.0

	// Output gcode header
	fmt.Println("G21 (metric ftw)")
	fmt.Println("G90 (absolute mode)")
	fmt.Printf("G92 X%.2f Y%.2f Z%.2f (you are here)\n", xHome, yHome, zHeight)
	fmt.Printf("G0 F%0.2f (Travel Feed Rate)\n", xyTravelRate)
	fmt.Printf("G1 F%0.2f (Cut Feed Rate)\n", xyFeedRate)
	fmt.Printf("G0 F%0.2f (Z Feed Rate)\n", zFeedRate)
	fmt.Printf("S%.2f (Pen Up Speed)\n", penUpSpeed)
	fmt.Println("M3 (Start cutter)")
	fmt.Printf("G0 Z%.2f (Pen Up)\n", penUpZ)

	// Iterate through paths, generating gcode for each
	transformY(svg, svg.HeightInMM())
	iterate(svg)

	// Output gcode footer
	fmt.Println()
	fmt.Println("(end of print job)")
	fmt.Printf("S%.2f (pen up speed)\n", penUpSpeed)
	fmt.Printf("G0 Z%.2f\n", penUpZ)
	fmt.Println("M5 (Stop cutter)")
	fmt.Printf("G0 X%0.2F Y%0.2F F%0.2F (go home)\n", xHome, yHome, xyTravelRate)
	fmt.Printf("G0 Z%0.2F F%02F\n", zHeight, zFeedRate)
}

// transformY transforms the Y coordinate system from SVG's positive Y = down to Timsav's positive Y = up
func transformY(svg *cleaner.SVGXMLNode, pageHeight float64) {
	// The timSave gcode extension somehow offsets the Y values by a few mm beyond the page height.
	// TODO: this shouldn't be necessary for the final extension, but it provides a useful equivalence during development.
	yOffset := pageHeight - 5.135

	for _, child := range svg.Children {
		for _, path := range child.Path {
			path.Y = yOffset - path.Y
			for _, drawTo := range path.DrawTo {
				drawTo.Y = yOffset - drawTo.Y
			}
		}
		transformY(child, pageHeight)
	}
}

func iterate(svg *cleaner.SVGXMLNode) {
	for _, child := range svg.Children {
		for _, path := range child.Path {
			// This gcode comment isn't terribly useful, but for now I'm just aiming for parity with the python extension.
			fmt.Println("\n(Polyline consisting of 1 segments.)")

			fmt.Printf("G0 X%0.2f Y%0.2f \n", path.X, path.Y)

			//z := 0.0
			switch child.Category {
			case cleaner.CategoryFullCut:
				fmt.Printf("G0 Z%0.2f S3800.00 (pen down through)\n", -8.0)
				//z = -8.0
			case cleaner.CategoryScore:
				fmt.Printf("G0 Z%0.2f S3800.00 (pen down score)\n", -5.0)
				//z = -5.0
			case cleaner.CategoryPaperCut:
				fmt.Printf("G0 Z%0.2f S3800.00 (pen down draw)\n", -2.0)
				//z = -2.0
			}

			//fmt.Printf("G0 Z%0.2f S3800.00\n", z)
			for _, drawTo := range path.DrawTo {
				switch drawTo.Command {
				case svgpath.LineTo:
					fmt.Printf("G1 X%0.2f Y%0.2f \n", drawTo.X, drawTo.Y)
				case svgpath.CurveTo:
					fmt.Printf("G5 X%0.2f Y%0.2f I%0.2f J%0.2f P%0.2f Q%0.2f \n", drawTo.X, drawTo.Y, drawTo.X1, drawTo.Y1, drawTo.X2, drawTo.Y2)
				}
			}

			fmt.Printf("G0 Z2.00 S3300.00 (Pen Up)\n")
		}
		iterate(child)
	}
}
