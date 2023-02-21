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
	X1 float64
	X2 float64
}

type testRunHandler struct {
	addRun    func(run *vectorize.Run)
	nextMinor func()
}

func (r *testRunHandler) AddRun(run *vectorize.Run) {
	r.addRun(run)
}

func (r *testRunHandler) NextY() {
	r.nextMinor()
}

func (r *testRunHandler) JoinerLines() []vectorize.Blob {
	return nil
}

func TestRunDetection(t *testing.T) {
	test := func(img *vectorize.ColorImage, expectedRuns []testRun) {
		i := 0
		minor := 0
		r := testRunHandler{
			addRun: func(run *vectorize.Run) {
				if i >= len(expectedRuns) {
					t.Fatalf("unexpected extra run")
				}
				diff := cmp.Diff(expectedRuns[i], testRun{X1: x1, X2: x2})
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
	), []testRun{})
}

func xTestLineDetection(t *testing.T) {
	test := func(img *vectorize.ColorImage) {
		pj := vectorize.NewBlobFinder(10, img.Width, img.Height)
		vectorize.FindHorizontalRuns(img, pj)
		lines := pj.Blobs()
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

	//img := loadImage("testdata/rough_diagonal.png")
	img := loadImage("testdata/test_transpose.png")
	fmt.Println(vectorize.Vectorize(img))
}
