package nrgba

import (
	"image/color"
	"math"

	"github.com/loov/imago/chroma"
)

// Linear is a premultiplied linear-light color with alpha, the working
// representation for blending 8-bit colors. It is float32 so per-frame code
// never converts precision; conversions from 8-bit go through a lookup table.
type Linear struct {
	R, G, B, A float32
}

// srgb8ToLinear maps an 8-bit sRGB component to linear light.
var srgb8ToLinear = func() (t [256]float32) {
	for i := range t {
		t[i] = float32(chroma.ToLinear(float64(i) / 255))
	}
	return
}()

// ToLinear converts a non-premultiplied sRGB color to premultiplied linear.
func ToLinear(c color.NRGBA) Linear {
	a := float32(c.A) / 0xFF
	return Linear{R: srgb8ToLinear[c.R] * a, G: srgb8ToLinear[c.G] * a, B: srgb8ToLinear[c.B] * a, A: a}
}

// NRGBA converts back to non-premultiplied sRGB; alpha 0 yields the zero color.
func (c Linear) NRGBA() color.NRGBA {
	if c.A == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{R: to8(toSRGB32(c.R / c.A)), G: to8(toSRGB32(c.G / c.A)), B: to8(toSRGB32(c.B / c.A)), A: to8(c.A)}
}

// Luminance is the relative luminance (Rec. 709 weights): 0 for black, 1 for white.
func (c Linear) Luminance() float32 { return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B }

// toSRGB32 is chroma.ToSRGB in float32 (EXT_sRGB formula), clamped to 0..1.
func toSRGB32(c float32) float32 {
	switch {
	case c <= 0:
		return 0
	case c < 0.0031308:
		return 12.92 * c
	case c < 1:
		return 1.055*float32(math.Pow(float64(c), 1/2.4)) - 0.055
	}
	return 1
}

// toLinear32 is chroma.ToLinear in float32.
func toLinear32(c float32) float32 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return float32(math.Pow(float64((c+0.055)/1.055), 2.4))
}

func to8(v float32) uint8 { return uint8(min(max(v, 0), 1)*255 + 0.5) }

// Opaque returns the color with alpha set to 1.
func (c Linear) Opaque() Linear {
	c.A = 1
	return c
}
