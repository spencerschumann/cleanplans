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
		X     float32
		Width int
	}

	test := func(img *vectorize.ColorImage, expectedRuns []Run) {
		i := 0
		vectorize.FindRuns(img, func(x float32, width int) {
			if i >= len(expectedRuns) {
				t.Fatalf("unexpected extra run")
			}
			diff := cmp.Diff(expectedRuns[i], Run{X: x, Width: width})
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
		"◻◻◻◻◻◻◻◻", // todo: need to add Y, and also relabel to major/minor, to verify that this row was skipped.
		"◼◼◼◼◼◼◼◼",
		"◼◼◻◻◻◻◼◼",
	), []Run{
		{X: 6, Width: 4},
		{X: 6, Width: 4},
		{X: 4, Width: 4},
		{X: 2, Width: 4},
		{X: 2, Width: 4},
		{X: 4, Width: 8},
		{X: 1, Width: 2}, {X: 7, Width: 2},
	})
}
