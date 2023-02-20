//go:build js && wasm

package main

import (
	"bytes"
	"cleanplans/pkg/color"
	"cleanplans/pkg/vectorize"
	"fmt"
	"image/png"
	"runtime"
	"time"

	"github.com/gopherjs/gopherjs/js"
)

// TODO: try running this code via GopherJS. How does it compare in size and speed
// against the same code compiled with the Go and/or TinyGo WASM targets?
//
// Why would I choose this approach? Well, I'm choosing Go for this project
// for several reasons, and I want to be able to run the same code in the browser,
// or as an inkscape extension on windows, mac, and linux, and maybe in other
// places in the future. Go's self-contained executables make this a breeze.

func main() {
	fmt.Println("GO MAIN CALLED!!!!!!!!!!!!")
	funcs := []struct {
		name string
		fn   any
	}{
		{"goCleanPlans", goCleanPlans},
	}
	for _, fn := range funcs {
		js.Global.Set(fn.name, fn.fn)
	}

	//<-make(chan any, 0)
}

func timeDeltaMS(t1, t2 time.Time) float64 {
	return float64(t2.Sub(t1)) / float64(time.Millisecond)
}

// goCleanPlans is the main entry point to cleanplans from JavaScript.
func goCleanPlans(image []byte, width, height, bitsPerPixel int) any {
	//image := args[0]

	//imageLen := image.Length()
	//width := args[1].Int()
	//height := args[2].Int()
	//bitsPerPixel := args[3].Int()
	imageLen := len(image)
	fmt.Printf("Go CleanPlans called: %d bytes, %d, %d, %d\n", imageLen, width, height, bitsPerPixel)

	t1 := time.Now()

	/*data := make([]byte, imageLen)
	js.CopyBytesToGo(data, image)*/
	//fmt.Println("Image:", image.String())
	//spew.Dump(image)
	//data := image.Interface().([]uint8)
	data := image

	ci := vectorize.PDFJSImageToColorImage(data, width, height, bitsPerPixel)
	t2 := time.Now()
	fmt.Printf("Created ColorImage %p, data=%p, width=%d, height=%d\n", ci, ci.Data, ci.Width, ci.Height)

	fmt.Printf("Time to run PDFJSImageToColorImage: %g\n", timeDeltaMS(t1, t2))

	if ci.Width < 100 && ci.Height < 100 {
		// For debugging, output an image that can be fed into tests
		str := "makeImage(\n"
		for y := 0; y < ci.Height; y++ {
			str += "    \""
			for x := 0; x < ci.Width; x++ {
				if ci.Data[x+y*ci.Width] == color.White {
					str += "◻"
				} else {
					str += "◼"
				}
			}
			str += "\",\n"
		}
		str += ")"
		fmt.Println(str)
	}

	// drop the data array now and run a GC cycle to conserve memory
	data = nil
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Println("Stats (sys/mallocs):", stats.Sys, stats.Mallocs)

	// Let's do a histogram of ci for debug.
	/*hist := make([]int, 9)
	for _, c := range ci.Data {
		hist[c] += 1
	}
	for k, v := range hist {
		fmt.Printf("Color histogram slice: %d %d\n", k, v)
	}*/

	svg := vectorize.Vectorize(ci)
	t3 := time.Now()
	fmt.Printf("Time to run Vectorize(): %g\n", timeDeltaMS(t2, t3))

	var buf bytes.Buffer
	err := png.Encode(&buf, ci)
	if err != nil {
		fmt.Printf("Error generating png image: %s\n", err)
		return nil
	}

	/*u8Array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(u8Array, buf.Bytes())*/

	t4 := time.Now()
	fmt.Printf("Time to encode png: %g\n", timeDeltaMS(t3, t4))

	return map[string]any{
		"png": buf.Bytes(),
		"svg": svg,
	}
}
