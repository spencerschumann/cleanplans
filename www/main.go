//go:build js && wasm

package main

import (
	"bytes"
	"cleanplans/pkg/vectorize"
	"fmt"
	"image/png"
	"runtime"
	"syscall/js"
)

func main() {
	js.Global().Set("goCleanPlans", js.FuncOf(goCleanPlans))
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

	var buf bytes.Buffer
	err := png.Encode(&buf, ci)
	if err != nil {
		fmt.Printf("Error generating png image: %s\n", err)
		return nil
	}

	fmt.Printf("Encoded png, %d bytes\n", buf.Len())

	svg := vectorize.Vectorize(ci)

	u8Array := js.Global().Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(u8Array, buf.Bytes())

	return map[string]any{
		"png": u8Array,
		"svg": svg,
	}
}
