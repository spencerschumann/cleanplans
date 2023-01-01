package main

//build: wasm

import (
	"cleanplans/pkg/color"
	"cleanplans/pkg/vectorize"
	"fmt"
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

	// Let's do a histogram of ci for debug.
	hist := map[color.Color]int{}
	for _, c := range ci {
		hist[c] += 1
	}
	for k, v := range hist {
		fmt.Printf("Color histogram: %d %d\n", k, v)
	}

	return nil
}
