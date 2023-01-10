package geometry

import (
	"math"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindArc(t *testing.T) {
	tests := []struct {
		start, mid, end Point
		want            Arc
	}{
		{
			start: Point{X: 0, Y: 0},
			mid:   Point{X: 1, Y: 1},
			end:   Point{X: 2, Y: 0},
			want: Arc{
				Start:     Point{X: 0, Y: 0},
				End:       Point{X: 2, Y: 0},
				Center:    Point{X: 1, Y: 0},
				Clockwise: true,
			},
		},
		{
			start: Point{X: 2, Y: 0},
			mid:   Point{X: 1, Y: 1},
			end:   Point{X: 0, Y: 0},
			want: Arc{
				Start:     Point{X: 2, Y: 0},
				End:       Point{X: 0, Y: 0},
				Center:    Point{X: 1, Y: 0},
				Clockwise: false,
			},
		},
		{
			start: Point{X: 0, Y: 0},
			mid:   Point{X: 0, Y: 0},
			end:   Point{X: 0, Y: 0},
			want: Arc{
				Start:     Point{X: 0, Y: 0},
				End:       Point{X: 0, Y: 0},
				Center:    Point{X: math.NaN(), Y: math.NaN()},
				Clockwise: false,
			},
		},
		{
			start: Point{X: 1.5, Y: 1.5},
			mid:   Point{X: 1.5, Y: 1.5},
			end:   Point{X: 1.5, Y: 1.5},
			want: Arc{
				Start:     Point{X: 1.5, Y: 1.5},
				End:       Point{X: 1.5, Y: 1.5},
				Center:    Point{X: math.NaN(), Y: math.NaN()},
				Clockwise: false,
			},
		},
		{
			start: Point{X: 0, Y: 0},
			mid:   Point{X: 1, Y: 0},
			end:   Point{X: 2, Y: 0},
			want: Arc{
				Start:     Point{X: 0, Y: 0},
				End:       Point{X: 2, Y: 0},
				Center:    Point{X: math.NaN(), Y: math.NaN()},
				Clockwise: false,
			},
		},
		{
			start: Point{X: 0, Y: 0},
			mid:   Point{X: -1, Y: 0},
			end:   Point{X: 2, Y: 0},
			want: Arc{
				Start:     Point{X: 0, Y: 0},
				End:       Point{X: 2, Y: 0},
				Center:    Point{X: math.NaN(), Y: math.NaN()},
				Clockwise: false,
			},
		},
		{
			start: Point{X: 0, Y: 0},
			mid:   Point{X: 2, Y: 2},
			end:   Point{X: 0, Y: 2},
			want: Arc{
				Start:     Point{X: 0, Y: 0},
				End:       Point{X: 0, Y: 2},
				Center:    Point{X: 1, Y: 1},
				Clockwise: false,
			},
		},
	}

	opt := cmp.Comparer(func(x, y float64) bool {
		nanX, nanY := math.IsNaN(x), math.IsNaN(y)
		if nanX != nanY {
			return false
		}
		if nanX && nanY {
			return true
		}
		return math.Abs(x-y) < 0.00001
	})

	for i, test := range tests {
		got := FindArc(test.start, test.mid, test.end)
		if diff := cmp.Diff(test.want, got, opt); diff != "" {
			t.Errorf("Test %d - FindArc(%v, %v, %v) incorrect output: %s", i, test.start, test.mid, test.end, diff)
		}
	}
}

func TestDouglasPeucker(t *testing.T) {
	tests := []struct {
		points     []Point
		epsilon    float64
		simplified []any
	}{
		{
			points: []Point{
				{0, 0},
				{1, 1},
				{2, 2},
				{3, 3},
				{4, 2},
				{5, 1},
				{6, 0},
			},
			epsilon: 0.001,
			simplified: []any{
				LineSegment{A: Point{X: 0, Y: 0}, B: Point{X: 3, Y: 3}},
				LineSegment{A: Point{X: 3, Y: 3}, B: Point{X: 6, Y: 0}},
			},
		},
		{
			points: []Point{
				{0, 0},
				{1, 0},
				{2, 0},
				{3, 0},
				{4, 0},
				{5, 0},
				{6, 0},
			},
			epsilon: 0.001,
			simplified: []any{
				LineSegment{A: Point{X: 0, Y: 0}, B: Point{X: 6, Y: 0}},
			},
		},
		{
			points: []Point{
				{0, 0},
				{1, 1},
				{2, 2},
				{3, 3},
				{4, 2},
				{5, 1},
				{6, 0},
			},
			epsilon: 1.5,
			simplified: []any{
				Arc{
					Start:     Point{X: 0, Y: 0},
					End:       Point{X: 6, Y: 0},
					Center:    Point{X: 3, Y: 0},
					Clockwise: true,
				},
			},
		},
		{
			points: []Point{
				{0, 0},
				{1, 1},
				{math.Sqrt(9.0 / 2.0), math.Sqrt(9.0 / 2.0)},
				{3, 3},
				{2 + math.Sqrt(9.0/2.0), math.Sqrt(9.0 / 2.0)},
				{5, 1},
				{6, 0},
			},
			epsilon: .01,
			simplified: []any{
				Arc{
					Start:     Point{X: 0, Y: 0},
					End:       Point{X: 6, Y: 0},
					Center:    Point{X: 3, Y: 0},
					Clockwise: true,
				},
			},
		},
		{
			points: []Point{
				{0, -4},
				{0, -3},
				{0, -2},
				{0, -1},
				{0, 0},
				{1, 1},
				{math.Sqrt(9.0 / 2.0), math.Sqrt(9.0 / 2.0)},
				{3, 3},
				{2 + math.Sqrt(9.0/2.0), math.Sqrt(9.0 / 2.0)},
				{5, 1},
				{6, 0},
			},
			epsilon: .01,
			simplified: []any{
				Arc{
					Start:     Point{X: 0, Y: 0},
					End:       Point{X: 6, Y: 0},
					Center:    Point{X: 3, Y: 0},
					Clockwise: true,
				},
			},
		},
	}
	for _, test := range tests {
		simplified := DouglasPeucker(test.points, test.epsilon)
		if !reflect.DeepEqual(simplified, test.simplified) {
			t.Errorf("DouglasPeucker(%v, %f) = %+v, want %+v", test.points, test.epsilon, simplified, test.simplified)
		}
	}
}
