package svgpath_test

import (
	"cleanplans/pkg/svgpath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBasic(t *testing.T) {
	subPaths, err := svgpath.Parse(" \t\r\nM1.e2 2. 1 .2.3 0.4e2 z L 7 8 9 10 H 11 12 13 L 2 2v5C 5 6 7 8 9 10")
	if err != nil {
		t.Errorf("parsing failed: %s", err)
	}
	expected := []*svgpath.SubPath{
		{X: 100, Y: 2, DrawTo: []*svgpath.DrawTo{
			{Command: svgpath.LineTo, X: 1, Y: .2},
			{Command: svgpath.LineTo, X: .3, Y: 40},
			{Command: svgpath.ClosePath, X: 100, Y: 2},
		}},
		{X: 100, Y: 2, DrawTo: []*svgpath.DrawTo{
			{Command: svgpath.LineTo, X: 7, Y: 8},
			{Command: svgpath.LineTo, X: 9, Y: 10},
			{Command: svgpath.LineTo, X: 11, Y: 10},
			{Command: svgpath.LineTo, X: 12, Y: 10},
			{Command: svgpath.LineTo, X: 13, Y: 10},
			{Command: svgpath.LineTo, X: 2, Y: 2},
			{Command: svgpath.LineTo, X: 2, Y: 7},
			{Command: svgpath.CurveTo, X: 9, Y: 10, X1: 5, Y1: 6, X2: 7, Y2: 8},
		}},
	}
	if diff := cmp.Diff(expected, subPaths); diff != "" {
		t.Errorf("incorrect output: %s", diff)
	}
}
