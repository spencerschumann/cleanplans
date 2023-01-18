package vectorize_test

import (
	"embed"
	"fmt"
	"image/png"
	"testing"
	"unicode/utf8"

	"cleanplans/pkg/color"
	"cleanplans/pkg/vectorize"

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
		vectorize.FindHorizontalRuns(img, &r)
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
		pj := vectorize.NewPointJoiner(10, img.Width, 1)
		vectorize.FindHorizontalRuns(img, pj)
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

//go:embed testdata/*.png
var pngData embed.FS

func loadImage(path string) *vectorize.ColorImage {
	file, _ := pngData.Open(path)
	defer file.Close()
	pngImg, _ := png.Decode(file)
	img := vectorize.ColorImage{
		Width:  pngImg.Bounds().Max.X,
		Height: pngImg.Bounds().Max.Y,
	}
	data := make([]color.Color, img.Width*img.Height)
	j := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			// TODO: if I need to read large images, this will probably be inefficient.
			// Better to access image bytes directly based on common types.
			r, g, b, _ := pngImg.At(x, y).RGBA()
			data[j] = color.RemapColor(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			j++
		}
	}
	img.Data = data
	return &img
}

func TestLargerImage(t *testing.T) {

	// TODO: need more tests with images with more noise and jitter.
	// TODO: need better handling of ends of lines, with their frequent
	// "hooks". Need to steer the lines toward minimizing error rather
	// than just picking a start and end point for a line segment and
	// calling it good if all segments are within the error range.
	// TODO: need to increase allowance for line histerisys - my
	// eye really wants to connect the line segments when I inspect
	// the results, but the current algorithms here break them up if
	// even one pixel is misplaced.

	img := loadImage("testdata/rough_diagonal.png")
	fmt.Println(vectorize.Vectorize(img))
}
