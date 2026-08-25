package nrgba

import "image/color"

// Lerp interpolates from c (p=0) to o (p=1), clamping p to 0..1.
func (c Linear) Lerp(o Linear, p float32) Linear {
	p = min(max(p, 0), 1)
	return Linear{R: c.R + (o.R-c.R)*p, G: c.G + (o.G-c.G)*p, B: c.B + (o.B-c.B)*p, A: c.A + (o.A-c.A)*p}
}

// MulAlpha scales all four premultiplied components by a.
func (c Linear) MulAlpha(a float32) Linear {
	return Linear{R: c.R * a, G: c.G * a, B: c.B * a, A: c.A * a}
}

// Over composites src over c (source-over, premultiplied).
func (c Linear) Over(src Linear) Linear {
	k := 1 - src.A
	return Linear{R: c.R*k + src.R, G: c.G*k + src.G, B: c.B*k + src.B, A: c.A*k + src.A}
}

// Over composites src over dst in linear light.
func Over(dst, src color.NRGBA) color.NRGBA { return ToLinear(dst).Over(ToLinear(src)).NRGBA() }

// Contrast is the WCAG contrast ratio (≥ 1) between the colors, ignoring alpha.
func Contrast(a, b color.NRGBA) float32 {
	la, lb := ToLinear(RGB8(a.R, a.G, a.B)).Luminance(), ToLinear(RGB8(b.R, b.G, b.B)).Luminance()
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// TextOn returns black or white, whichever contrasts more with bg.
func TextOn(bg color.NRGBA) color.NRGBA {
	black, white := Black, White
	if Contrast(bg, black) >= Contrast(bg, white) {
		return black
	}
	return white
}

// Ramp returns n ≥ 2 stops from a to b, interpolated in linear light; nil for n < 2.
func Ramp(a, b color.NRGBA, n int) []color.NRGBA {
	if n < 2 {
		return nil
	}
	la, lb := ToLinear(a), ToLinear(b)
	out := make([]color.NRGBA, n)
	for i := range out {
		out[i] = la.Lerp(lb, float32(i)/float32(n-1)).NRGBA()
	}
	return out
}
