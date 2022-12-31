package main

//build: wasm

import (
	"fmt"
	"syscall/js"
	"time"
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
	imageType := args[3].Int()
	fmt.Printf("Go CleanPlans called: %d bytes, %d, %d, %d\n", imageLen, width, height, imageType)

	data := make([]byte, imageLen)
	js.CopyBytesToGo(data, image)

	var average float64

	for loops := 0; loops < 10; loops++ {
		start := time.Now()

		var total float64
		for i := 0; i < imageLen; i++ {
			total += float64(data[i])
		}
		average = total / float64(len(data))
		fmt.Printf("Average value: %f\n", average)

		dur := time.Since(start)
		fmt.Printf("Time to process data using js.CopyBytesToGo method: %s\n", dur)
	}

	return average
}
