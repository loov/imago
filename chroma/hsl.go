package chroma

import "math"

// HSL is hue/saturation/lightness over encoded SRGB. H in turns [0, 1).
type HSL struct{ H, S, L float64 }

// HSV is hue/saturation/value over encoded SRGB. H in turns [0, 1).
type HSV struct{ H, S, V float64 }

// hueChroma returns hue in turns, max, and chroma of an SRGB color.
func hueChroma(c SRGB) (h, mx, ch float64) {
	mx = max(c.R, c.G, c.B)
	ch = mx - min(c.R, c.G, c.B)
	if ch == 0 {
		return 0, mx, 0
	}
	switch mx {
	case c.R:
		h = math.Mod((c.G-c.B)/ch, 6)
	case c.G:
		h = (c.B-c.R)/ch + 2
	default:
		h = (c.R-c.G)/ch + 4
	}
	h /= 6
	if h < 0 {
		h++
	}
	return h, mx, ch
}

// hueRGB builds SRGB from hue, chroma and the minimum component m.
func hueRGB(h, ch, m float64) SRGB {
	h = math.Mod(h, 1)
	if h < 0 {
		h++
	}
	h *= 6
	x := ch * (1 - math.Abs(math.Mod(h, 2)-1))
	var r, g, b float64
	switch int(h) {
	case 0:
		r, g = ch, x
	case 1:
		r, g = x, ch
	case 2:
		g, b = ch, x
	case 3:
		g, b = x, ch
	case 4:
		r, b = x, ch
	default:
		r, b = ch, x
	}
	return SRGB{r + m, g + m, b + m}
}

// HSLFromSRGB converts encoded sRGB to HSL.
func HSLFromSRGB(c SRGB) HSL {
	h, mx, ch := hueChroma(c)
	l := mx - ch/2
	var s float64
	if ch != 0 && l != 0 && l != 1 {
		s = ch / (1 - math.Abs(2*l-1))
	}
	return HSL{h, s, l}
}

// SRGB converts HSL to encoded sRGB.
func (c HSL) SRGB() SRGB {
	ch := (1 - math.Abs(2*c.L-1)) * c.S
	return hueRGB(c.H, ch, c.L-ch/2)
}

// HSVFromSRGB converts encoded sRGB to HSV.
func HSVFromSRGB(c SRGB) HSV {
	h, mx, ch := hueChroma(c)
	var s float64
	if mx != 0 {
		s = ch / mx
	}
	return HSV{h, s, mx}
}

// SRGB converts HSV to encoded sRGB.
func (c HSV) SRGB() SRGB {
	ch := c.V * c.S
	return hueRGB(c.H, ch, c.V-ch)
}
