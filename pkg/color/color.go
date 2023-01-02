package color

import "image/color"

// Color is a restricted, cannonical color palette for plans.
// Its intent is to cover every color with a distinct meaning
// in plans, with a minimal number of colors that are visually
// as distinct as possible.
// This is very similar to the 3-bit RGB palette, described at
// https://en.wikipedia.org/wiki/List_of_monochrome_and_RGB_color_formats#3-bit_RGB,
// but with the following changes:
// * Gray as an added color (in the center of the color cube)
// * Yellow shifted toward orange to better distinguish it from white
// * Cyan darkened and shifted toward blue to better distinguish it from green
// * Magenta darkened to better distinguish it from red
//
// A dark yellow and/or brown might be useful to add. This
// palette currently has 9 colors; it could be expanded to
// 16 and fit 2 pixels per byte, although for now I'm
// just keeping it simple and using one byte per pixel.
type Color byte

const (
	White Color = iota
	Black
	Gray
	Red
	Green
	Blue
	Magenta
	Cyan
	Orange
)

var Palette = color.Palette{
	color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // White
	color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // Black
	color.RGBA{R: 0x7f, G: 0x7f, B: 0x7f, A: 0xff}, // Gray
	color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff}, // Red
	color.RGBA{R: 0x00, G: 0xcc, B: 0x00, A: 0xff}, // Green
	color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // Blue
	color.RGBA{R: 0xcc, G: 0x00, B: 0xcc, A: 0xff}, // Magenta
	color.RGBA{R: 0x00, G: 0xbb, B: 0xdd, A: 0xff}, // Cyan
	color.RGBA{R: 0xff, G: 0xdd, B: 0x00, A: 0xff}, // Orange
}

func ColorToImageColor(c Color) color.Color {
	if int(c) >= len(Palette) {
		return Palette[White]
	}
	return Palette[c]
}

func min3(a, b, c byte) byte {
	if c < b {
		b = c
	}
	if b < a {
		a = b
	}
	return a
}

func max3(a, b, c byte) byte {
	if b < a {
		b = a
	}
	if c < b {
		c = b
	}
	return c
}

// RemapColor remaps an RGB color, expressed as r, g, and b components, to a Color.
func RemapColor(r, g, b byte) Color {
	// Check for pure white or pure black first - these are expected to be most common
	/*if r > 200 && g > 200 && b > 200 {
		return White
	}
	if r < 50 && g < 50 && b < 50 {
		return Black
	}*/

	min := min3(r, g, b)
	max := max3(r, g, b)
	lightness := max/2 + min/2 // divide each separately before adding to avoid byte overflow

	// TODO: configurable thresholds

	// Most pixels are expected to be white, so check for white first
	if lightness >= 192 {
		return White
	}

	// Black is expected to be second most common
	if lightness < 32 {
		return Black
	}

	chroma := max - min

	if lightness < 86 {
		// Note: this only checks this one pixel's chroma, but if the neighboring
		// chroma is high, this pixel could actually be intended to have color.
		// It's uncertain whether accounting for this would improve accuracy. The
		// commented block of code below shows how this can be done, if needed.
		if chroma <= max/2 {
			return Black
		}
	}

	/*
	   // JPEG compression smears the colors, so use an average of the
	   // neighboring colors rather than just the chroma of the pixel itself.
	   let points = [(x, y), (x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)];
	   let mut r = 0;
	   let mut g = 0;
	   let mut b = 0;
	   for (x, y) in points {
	       let rgb = getpx(x, y);
	       r += rgb[0] as i32;
	       g += rgb[1] as i32;
	       b += rgb[2] as i32;
	   }
	   let len = points.len() as i32;
	   let r = r / len;
	   let g = g / len;
	   let b = b / len;

	   // assume full saturation, with a lightness of 1/2
	   let min = min3(r, g, b);
	   let max = max3(r, g, b);
	   chroma = max - min;*/

	mid := min/2 + max/2

	if chroma < 8 {
		return Gray
	} else if r == max {
		if b < mid {
			// For yellow tones, shift slightly toward orange because
			// yellow is hard to distinguish from white
			if g < 90 {
				return Red
			} else {
				return Orange
			}
		} else {
			return Magenta
		}
	} else if g == max {
		if b < mid {
			if r < mid {
				return Green
			} else {
				return Orange
			}
		} else {
			return Cyan
		}
	} else {
		// Otherwise, blue is max
		if r < mid {
			if g < mid {
				return Blue
			} else {
				return Cyan
			}
		} else {
			return Magenta
		}
	}
}
