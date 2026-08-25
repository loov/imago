// Package nrgba is the per-frame toolkit for color.NRGBA: constructors,
// premultiplication, integer mixes and a float32 linear-light form. The
// per-frame paths (Linear, Lerp, Mix, Over, Premultiply) are float32; the
// theme-time adjustments (Lighten, Darken, Saturate, Shift, Gray) go
// through package chroma's float64 spaces.
//
// Build colors with RGB8, Hex or Parse, then combine them:
//
//	bg := nrgba.Hex(0x1e1e2e)
//	fg := nrgba.TextOn(bg)                   // black or white, whichever contrasts more
//	hover := nrgba.Lighten(bg, 0.1)          // OkLCh lightness step
//	mixed := nrgba.Over(bg, nrgba.MulAlpha(fg, 128))
//
// Blending in sRGB bytes darkens midtones. For correct blends convert to
// Linear, blend, and convert back:
//
//	c := nrgba.ToLinear(a).Lerp(nrgba.ToLinear(b), 0.5).NRGBA()
//
// Ramp builds n-step gradients for palettes; Contrast returns the WCAG
// contrast ratio.
package nrgba
