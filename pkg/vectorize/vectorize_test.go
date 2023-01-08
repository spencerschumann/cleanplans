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

type testRun struct {
	Center float32
	Minor  int
	Width  int
}

type testRunHandler struct {
	addRun    func(center float32, width int)
	nextMinor func()
}

func (r *testRunHandler) AddRun(center float32, width int) {
	r.addRun(center, width)
}

func (r *testRunHandler) NextMinor() {
	r.nextMinor()
}

func (r *testRunHandler) JoinerLines() []vectorize.JoinerLine {
	return nil
}

func TestRunDetection(t *testing.T) {
	test := func(img *vectorize.ColorImage, expectedRuns []testRun) {
		i := 0
		minor := 0
		r := testRunHandler{
			addRun: func(center float32, width int) {
				if i >= len(expectedRuns) {
					t.Fatalf("unexpected extra run")
				}
				diff := cmp.Diff(expectedRuns[i], testRun{Center: center, Minor: minor, Width: width})
				if diff != "" {
					t.Fatalf("Run index %d incorrect: %s", i, diff)
				}
				i++
			},
			nextMinor: func() {
				minor++
			},
		}
		vectorize.FindHorizontalRuns(img, 20, &r)
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
	), []testRun{
		{Center: 6, Minor: 0, Width: 4},
		{Center: 6, Minor: 1, Width: 4},
		{Center: 4, Minor: 2, Width: 4},
		{Center: 2, Minor: 3, Width: 4},
		{Center: 2, Minor: 4, Width: 4},
		{Center: 4, Minor: 6, Width: 8},
		{Center: 1, Minor: 7, Width: 2}, {Center: 7, Minor: 7, Width: 2},
	})
}

func xTestLineDetection(t *testing.T) {
	test := func(img *vectorize.ColorImage) {
		pj := vectorize.NewPointJoiner(10, img.Width)
		vectorize.FindHorizontalRuns(img, 20, pj)
		lines := pj.JoinerLines()
		t.Errorf("Lines: %#v\n", lines)
	}

	test(makeImage(
		"◻◻◻◻◼◼◻◻",
		"◻◻◻◼◼◻◻◻",
		"◻◻◼◼◻◻◻◻",
		"◻◼◼◻◻◻◻◻",
		"◼◼◻◻◻◻◻◻",
		"◻◻◻◻◻◻◻◻",
		"◼◼◼◼◼◼◼◼",
		"◻◻◻◻◻◻◻◻",
		"◻◻◼◼◼◼◻◻",
		"◻◻◼◼◼◼◻◻",
		"◻◻◼◼◼◼◻◻",
		"◻◻◼◼◼◼◻◻",
		"◻◻◻◻◻◻◻◻",
	))
}
