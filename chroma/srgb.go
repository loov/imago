package chroma

import "math"

// SRGB is gamma-encoded sRGB with components nominally in 0..1.
type SRGB struct{ R, G, B float64 }

// SRGBFrom8 converts 8-bit sRGB components.
func SRGBFrom8(r, g, b uint8) SRGB {
	return SRGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}
}

// To8 clamps and rounds to 8-bit components.
func (c SRGB) To8() (r, g, b uint8) {
	c = c.Clamp()
	return uint8(c.R*255 + 0.5), uint8(c.G*255 + 0.5), uint8(c.B*255 + 0.5)
}

// ToLinear applies the inverse sRGB transfer function to one component.
func ToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// ToSRGB applies the sRGB transfer function to one linear component.
func ToSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// RGB decodes to linear light.
func (c SRGB) RGB() RGB { return RGB{ToLinear(c.R), ToLinear(c.G), ToLinear(c.B)} }

// SRGBFromRGB encodes linear light.
func SRGBFromRGB(c RGB) SRGB { return SRGB{ToSRGB(c.R), ToSRGB(c.G), ToSRGB(c.B)} }

// Clamp limits each component to 0..1.
func (c SRGB) Clamp() SRGB {
	return SRGB{min(max(c.R, 0), 1), min(max(c.G, 0), 1), min(max(c.B, 0), 1)}
}
