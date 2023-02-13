//go:build js && wasm

package main

import (
	"bytes"
	"cleanplans/pkg/color"
	"cleanplans/pkg/vectorize"
	"fmt"
	"image/png"
	"runtime"
	"syscall/js"
)

// TODO: try running this code via GopherJS. How does it compare in size and speed
// against the same code compiled with the Go and/or TinyGo WASM targets?
//
// Why would I choose this approach? Well, I'm choosing Go for this project
// for several reasons, and I want to be able to run the same code in the browser,
// or as an inkscape extension on windows, mac, and linux, and maybe in other
// places in the future. Go's self-contained executables make this a breeze.

func main() {
	funcs := []struct {
		name string
		fn   func(js.Value, []js.Value) any
	}{
		{"goCleanPlans", goCleanPlans},
	}
	for _, fn := range funcs {
		js.Global().Set(fn.name, js.FuncOf(fn.fn))
	}

	<-make(chan any, 0)
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
	fmt.Printf("Created ColorImage %p, data=%p, width=%d, height=%d\n", ci, ci.Data, ci.Width, ci.Height)

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

	var buf bytes.Buffer
	err := png.Encode(&buf, ci)
	if err != nil {
		fmt.Printf("Error generating png image: %s\n", err)
		return nil
	}

	u8Array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(u8Array, buf.Bytes())

	return map[string]any{
		"png": u8Array,
		"svg": svg,
	}
}
