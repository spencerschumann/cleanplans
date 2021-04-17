package cleaner

import (
	"log"
	"math"
)

type Category int

const (
	CategoryNone Category = iota
	CategoryFullCut
	CategoryScore
	CategoryPaperCut
	CategoryOptional
	CategoryCrease
)

/*
	Black - full cut: rgb(0%,0%,0%) [rgb(0%,0%,0%) 0 0 0]
	Blue - crease: rgb(0%,0%,100%) [rgb(0%,0%,100%) 0 0 100]
	Green - ref/optional: rgb(0%,100%,0%) [rgb(0%,100%,0%) 0 100 0]
	Dark Blue - instructions: rgb(0%,43.920898%,58.430481%) [rgb(0%,43.920898%,58.430481%) 0 43.920898 58.430481]
	Cyan - bevel: rgb(0%,100%,100%) [rgb(0%,100%,100%) 0 100 100]
	Red - score: rgb(100%,0%,0%) [rgb(100%,0%,0%) 100 0 0]
	Magenta - remove paper: rgb(100%,0%,100%) [rgb(100%,0%,100%) 100 0 100]
	Orange - cavity: rgb(100%,49.803162%,0%) [rgb(100%,49.803162%,0%) 100 49.803162 0]

	Keep Black
	Keep Red (convert dashed lines to solid)

	Keep Magenta as light reference line

	Remove Blue (or keep as light reference line)
	Remove Green (or keep for reference)
	Remove Cyan (or keep dashed lines for reference)

	Remove Dark Blue
	Remove Magenta dots
	Remove Orange
*/

func colorsNear(a, b Color) bool {
	delta := math.Abs(a.R-b.R) +
		math.Abs(a.G-b.G) +
		math.Abs(a.B-b.B)
	return delta < 0.05 // TODO: configurable?
}

var (
	colorBlack   = Color{R: 0, G: 0, B: 0}
	colorRed     = Color{R: 1, G: 0, B: 0}
	colorMagenta = Color{R: 1, G: 0, B: 1}
	colorGreen   = Color{R: 0, G: 1, B: 0}
	colorBlue    = Color{R: 0, G: 0, B: 1}
)

func classifyStroke(stroke string) Category {
	if stroke == "" || stroke == "none" {
		return CategoryNone
	}

	c, err := parseColor(stroke)
	if err != nil {
		log.Fatalf("error parsing color: %s", err)
	}

	if colorsNear(c, colorBlack) {
		return CategoryFullCut
	}
	if colorsNear(c, colorRed) {
		return CategoryScore
	}
	if colorsNear(c, colorMagenta) {
		return CategoryPaperCut
	}
	if colorsNear(c, colorGreen) {
		return CategoryOptional
	}
	if colorsNear(c, colorBlue) {
		return CategoryCrease
	}
	return CategoryNone
}
