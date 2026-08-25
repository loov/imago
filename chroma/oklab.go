package chroma

import "math"

// Oklab is Björn Ottosson's perceptual space, L in 0..1.
// https://bottosson.github.io/posts/oklab/
type Oklab struct{ L, A, B float64 }

// Distance is the Euclidean distance ΔE_ok between two colors.
func (c Oklab) Distance(o Oklab) float64 {
	return math.Sqrt((c.L-o.L)*(c.L-o.L) + (c.A-o.A)*(c.A-o.A) + (c.B-o.B)*(c.B-o.B))
}

// OkLCh is Oklab in polar form, H in turns [0, 1).
type OkLCh struct{ L, C, H float64 }

var (
	// M1 and M2 from https://bottosson.github.io/posts/oklab/.
	oklabM1 = mat{
		{0.4122214708, 0.5363325363, 0.0514459929},
		{0.2119034982, 0.6806995451, 0.1073969566},
		{0.0883024619, 0.2817188376, 0.6299787005},
	}
	oklabM2 = mat{
		{0.2104542553, 0.7936177850, -0.0040720468},
		{1.9779984951, -2.4285922050, 0.4505937099},
		{0.0259040371, 0.7827717662, -0.8086757660},
	}
	oklabM1Inv = oklabM1.inv()
	oklabM2Inv = oklabM2.inv()
)

// OklabFromRGB converts linear RGB to Oklab.
func OklabFromRGB(c RGB) Oklab {
	l, m, s := oklabM1.mul(c.R, c.G, c.B)
	L, A, B := oklabM2.mul(math.Cbrt(l), math.Cbrt(m), math.Cbrt(s))
	return Oklab{L, A, B}
}

// RGB converts Oklab to linear RGB.
func (c Oklab) RGB() RGB {
	l, m, s := oklabM2Inv.mul(c.L, c.A, c.B)
	r, g, b := oklabM1Inv.mul(l*l*l, m*m*m, s*s*s)
	return RGB{r, g, b}
}

// OkLChFromOklab converts to polar form.
func OkLChFromOklab(c Oklab) OkLCh {
	ch, h := polar(c.A, c.B)
	return OkLCh{c.L, ch, h}
}

// Oklab converts to rectangular form.
func (c OkLCh) Oklab() Oklab {
	a, b := rect(c.C, c.H)
	return Oklab{c.L, a, b}
}

// polar returns chroma and hue in turns [0, 1); hue is 0 when achromatic.
func polar(a, b float64) (c, h float64) {
	c = math.Hypot(a, b)
	if c == 0 {
		return 0, 0
	}
	h = math.Atan2(b, a) / (2 * math.Pi)
	if h < 0 {
		h++
	}
	if h >= 1 {
		h = 0
	}
	return c, h
}

func rect(c, h float64) (a, b float64) {
	r := h * 2 * math.Pi
	return c * math.Cos(r), c * math.Sin(r)
}

// inGamut reports whether the linear RGB equivalent is within 0..1 (tolerance 1e-4).
func (c OkLCh) inGamut() bool {
	r := c.Oklab().RGB()
	const lo, hi = -1e-4, 1 + 1e-4
	return r.R >= lo && r.R <= hi && r.G >= lo && r.G <= hi && r.B >= lo && r.B <= hi
}

// Clamp reduces chroma until in sRGB gamut, keeping L and H fixed.
// L outside 0..1 cannot be mapped and is returned with C = 0.
func (c OkLCh) Clamp() OkLCh {
	if c.inGamut() {
		return c
	}
	lo, hi := 0.0, c.C
	for i := 0; i < 20; i++ {
		if mid := (lo + hi) / 2; (OkLCh{c.L, mid, c.H}).inGamut() {
			lo = mid
		} else {
			hi = mid
		}
	}
	return OkLCh{c.L, lo, c.H}
}
