// Package nrgba is the per-frame toolkit for color.NRGBA: constructors,
// premultiplication, integer mixes and a float32 linear-light form. It
// never uses float64; package chroma holds the exact color-space math.
package nrgba

import (
	"image/color"

	"github.com/loov/imago/chroma"
)

// Hex returns the color for 0xRRGGBBAA.
func Hex(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 24), G: uint8(hex >> 16), B: uint8(hex >> 8), A: uint8(hex)}
}

// FromSRGB quantizes to an opaque 8-bit color, clamping to gamut.
func FromSRGB(c chroma.SRGB) color.NRGBA {
	r, g, b := c.To8()
	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}
}

// SRGB converts the color components to chroma.SRGB, ignoring alpha.
func SRGB(c color.NRGBA) chroma.SRGB { return chroma.SRGBFrom8(c.R, c.G, c.B) }

// Lerp linearly interpolates each component from a (p=0) to b (p=1),
// clamping p to 0..1, in encoded sRGB.
func Lerp(a, b color.NRGBA, p float32) color.NRGBA {
	p = min(max(p, 0), 1)
	mix := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*p + 0.5) }
	return color.NRGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: mix(a.A, b.A)}
}

// Mix mixes a and b weighted by t/256 and (1 - t/256) respectively.
func Mix(a, b color.NRGBA, t uint8) color.NRGBA {
	ti := int(t)
	return color.NRGBA{
		R: byte((int(a.R)*ti + int(b.R)*(256-ti)) / 256),
		G: byte((int(a.G)*ti + int(b.G)*(256-ti)) / 256),
		B: byte((int(a.B)*ti + int(b.B)*(256-ti)) / 256),
		A: byte((int(a.A)*ti + int(b.A)*(256-ti)) / 256),
	}
}

// MulAlpha multiplies the color's alpha by alpha/255.
func MulAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
	c.A = uint8(uint32(c.A) * uint32(alpha) / 0xFF)
	return c
}

// Premultiply converts non-premultiplied sRGB to premultiplied sRGB.
// Multiplication happens in linear light, so each result component is
// chroma.ToSRGB(chroma.ToLinear(c) * alpha), computed in float32.
func Premultiply(c color.NRGBA) color.RGBA {
	if c.A == 0xFF {
		return color.RGBA(c)
	}
	l := ToLinear(c)
	return color.RGBA{R: to8(toSRGB32(l.R)), G: to8(toSRGB32(l.G)), B: to8(toSRGB32(l.B)), A: c.A}
}

// LinearPremultiply converts non-premultiplied sRGB to premultiplied 8-bit
// linear RGBA: each component is chroma.ToLinear(c) * alpha.
func LinearPremultiply(c color.NRGBA) color.RGBA {
	if c.A == 0xFF {
		return color.RGBA(c)
	}
	l := ToLinear(c)
	return color.RGBA{R: to8(l.R), G: to8(l.G), B: to8(l.B), A: c.A}
}

// Unpremultiply converts premultiplied sRGB to non-premultiplied sRGB.
func Unpremultiply(c color.RGBA) color.NRGBA {
	if c.A == 0xFF {
		return color.NRGBA(c)
	}
	return Linear{
		R: toLinear32(float32(c.R) / 0xFF),
		G: toLinear32(float32(c.G) / 0xFF),
		B: toLinear32(float32(c.B) / 0xFF),
		A: float32(c.A) / 0xFF,
	}.NRGBA()
}
