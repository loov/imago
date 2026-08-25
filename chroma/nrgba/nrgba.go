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
// t = 255 gives 255/256 of a, never exactly a; use Lerp for exact endpoints.
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

// Premultiply converts non-premultiplied sRGB to premultiplied sRGB in the
// encoded channel values; matches image/color's conversion.
func Premultiply(c color.NRGBA) color.RGBA {
	// Same arithmetic as color.NRGBA.RGBA followed by color.RGBAModel.
	a := uint32(c.A) * 0x101
	mul := func(v uint8) uint8 { return uint8(uint32(v) * 0x101 * a / 0xFFFF >> 8) }
	return color.RGBA{R: mul(c.R), G: mul(c.G), B: mul(c.B), A: c.A}
}

// Unpremultiply converts premultiplied sRGB to non-premultiplied sRGB in the
// encoded channel values; matches image/color's conversion. Alpha 0 yields
// the zero color.
func Unpremultiply(c color.RGBA) color.NRGBA {
	if c.A == 0 {
		return color.NRGBA{}
	}
	// Same arithmetic as color.RGBA.RGBA followed by color.NRGBAModel.
	a := uint32(c.A) * 0x101
	div := func(v uint8) uint8 { return uint8((uint32(v) * 0x101 * 0xFFFF / a) >> 8) }
	return color.NRGBA{R: div(c.R), G: div(c.G), B: div(c.B), A: c.A}
}
