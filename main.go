package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"cleanplans/pkg/cleaner"
	"cleanplans/pkg/gcode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s svg-file\n", os.Args[0])
		return
	}

	filename := os.Args[1]
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("file read error: %s", err)
	}

	svg, err := cleaner.Parse(data)
	if err != nil {
		log.Fatalf("parse error: %s", err)
	}

	svg.FilteredAbsoluteMM()
	svg.RotateAndCenter(
		508, // 20 inches in mm
		//762, // 30 inches in mm
		757, // My batch of foam board is actually 29 13/16 inches.
	)

	cleaner.Undash(svg)
	cleaner.Simplify(svg)

	gcode.Generate(svg)

	/*
		outXML, err := svg.Marshal()
		if err != nil {
			log.Fatalf("marshal error: %s", err)
		}
		fmt.Println(string(outXML))
	*/
}
