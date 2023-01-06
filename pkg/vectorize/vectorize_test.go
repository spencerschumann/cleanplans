package vectorize_test

import (
	"cleanplans/pkg/color"
	"cleanplans/pkg/vectorize"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

func makeImage(rows ...string) *vectorize.ColorImage {
	img := vectorize.ColorImage{
		Width:  utf8.RuneCountInString(rows[0]),
		Height: len(rows),
	}
	img.Data = make([]color.Color, img.Width*img.Height)
	i := 0
	for _, row := range rows {
		for _, ch := range row {
			if ch == '◻' {
				img.Data[i] = color.White
			} else if ch == '◼' {
				img.Data[i] = color.Black
			}
			i++
		}
	}
	return &img
}

func TestRunDetection(t *testing.T) {
	type Run struct {
		Center float32
		Minor  int
		Width  int
	}

	test := func(img *vectorize.ColorImage, expectedRuns []Run) {
		i := 0
		vectorize.FindRuns(img, func(center float32, minor int, width int) {
			if i >= len(expectedRuns) {
				t.Fatalf("unexpected extra run")
			}
			diff := cmp.Diff(expectedRuns[i], Run{Center: center, Minor: minor, Width: width})
			if diff != "" {
				t.Fatalf("Run index %d incorrect: %s", i, diff)
			}
			i++
		})
		if i != len(expectedRuns) {
			t.Fatalf("got fewer runs (%d) than expected (%d)", i, len(expectedRuns))
		}
	}

	test(makeImage(
		"◻◻◻◻◼◼◼◼",
		"◻◻◻◻◼◼◼◼",
		"◻◻◼◼◼◼◻◻",
		"◼◼◼◼◻◻◻◻",
		"◼◼◼◼◻◻◻◻",
		"◻◻◻◻◻◻◻◻",
		"◼◼◼◼◼◼◼◼",
		"◼◼◻◻◻◻◼◼",
	), []Run{
		{Center: 6, Minor: 0, Width: 4},
		{Center: 6, Minor: 1, Width: 4},
		{Center: 4, Minor: 2, Width: 4},
		{Center: 2, Minor: 3, Width: 4},
		{Center: 2, Minor: 4, Width: 4},
		{Center: 4, Minor: 6, Width: 8},
		{Center: 1, Minor: 7, Width: 2}, {Center: 7, Minor: 7, Width: 2},
	})
}
