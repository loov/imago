package nrgba

import (
	"image/color"
	"math"

	"github.com/loov/imago/chroma"
)

// adjust applies f in OkLCh, clamps to gamut and keeps the original alpha.
// It goes through float64 chroma; intended for theme setup, not per-pixel use.
func adjust(c color.NRGBA, f func(chroma.OkLCh) chroma.OkLCh) color.NRGBA {
	lch := f(chroma.OkLChFromOklab(chroma.OklabFromRGB(SRGB(c).RGB()))).Clamp()
	out := FromSRGB(chroma.SRGBFromRGB(lch.Oklab().RGB()))
	out.A = c.A
	return out
}

// Lighten raises OkLCh lightness by amount (clamped to 0..1). Theme-time; uses float64.
func Lighten(c color.NRGBA, amount float32) color.NRGBA {
	return adjust(c, func(l chroma.OkLCh) chroma.OkLCh {
		l.L = min(max(l.L+float64(amount), 0), 1)
		return l
	})
}

// Darken lowers OkLCh lightness by amount (clamped to 0..1). Theme-time; uses float64.
func Darken(c color.NRGBA, amount float32) color.NRGBA { return Lighten(c, -amount) }

// Saturate scales OkLCh chroma by 1+amount (amount may be negative). Theme-time; uses float64.
func Saturate(c color.NRGBA, amount float32) color.NRGBA {
	return adjust(c, func(l chroma.OkLCh) chroma.OkLCh {
		l.C = max(l.C*(1+float64(amount)), 0)
		return l
	})
}

// Shift rotates OkLCh hue by turns. Theme-time; uses float64.
func Shift(c color.NRGBA, turns float32) color.NRGBA {
	return adjust(c, func(l chroma.OkLCh) chroma.OkLCh {
		l.H = math.Mod(math.Mod(l.H+float64(turns), 1)+1, 1)
		return l
	})
}

// Gray removes chroma, keeping OkLCh lightness. Theme-time; uses float64.
func Gray(c color.NRGBA) color.NRGBA {
	return adjust(c, func(l chroma.OkLCh) chroma.OkLCh {
		l.C = 0
		return l
	})
}
