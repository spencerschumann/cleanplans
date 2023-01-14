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

// This one is useful enough to leave here; for now, just as a comment.
/*
func arcToPath(arc geometry.Arc) string {
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
}*/

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
