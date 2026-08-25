package nrgba

import (
	"image/color"
	"math"
)

// Common colors.
var (
	White       = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	Black       = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	Red         = color.NRGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}
	Green       = color.NRGBA{R: 0x00, G: 0xFF, B: 0x00, A: 0xFF}
	Blue        = color.NRGBA{R: 0x00, G: 0x00, B: 0xFF, A: 0xFF}
	Yellow      = color.NRGBA{R: 0xFF, G: 0xFF, B: 0x00, A: 0xFF}
	Transparent = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x00}
)

// RGB returns an opaque color from components in 0..1, saturating.
func RGB(r, g, b float32) color.NRGBA { return RGBA(r, g, b, 1) }

// RGBA returns a color from components in 0..1, saturating.
func RGBA(r, g, b, a float32) color.NRGBA {
	return color.NRGBA{R: sat8(r), G: sat8(g), B: sat8(b), A: sat8(a)}
}

// Floats returns the components scaled to 0..1.
func Floats(c color.NRGBA) (r, g, b, a float32) {
	return float32(c.R) / 0xFF, float32(c.G) / 0xFF, float32(c.B) / 0xFF, float32(c.A) / 0xFF
}

// HSL returns an opaque color from hue in turns (0 and 1 are both red),
// saturation and lightness in 0..1. Computed in float32; for the exact
// conversion use chroma.HSL.
func HSL(h, s, l float32) color.NRGBA { return HSLA(h, s, l, 1) }

// HSLA is HSL with alpha.
func HSLA(h, s, l, a float32) color.NRGBA {
	if s == 0 {
		return RGBA(l, l, l, a)
	}
	h = float32(math.Mod(float64(h), 1))
	if h < 0 {
		h++
	}
	var v2 float32
	if l < 0.5 {
		v2 = l * (1 + s)
	} else {
		v2 = l + s - s*l
	}
	v1 := 2*l - v2
	return RGBA(hue(v1, v2, h+1.0/3), hue(v1, v2, h), hue(v1, v2, h-1.0/3), a)
}

func hue(v1, v2, h float32) float32 {
	if h < 0 {
		h++
	} else if h > 1 {
		h--
	}
	switch {
	case 6*h < 1:
		return v1 + (v2-v1)*6*h
	case 2*h < 1:
		return v2
	case 3*h < 2:
		return v1 + (v2-v1)*(2.0/3-h)*6
	}
	return v1
}

func sat8(v float32) uint8 { return uint8(min(max(v, 0), 1)*255 + 0.5) }
