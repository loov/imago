// Package chroma implements float64 conversions between common color spaces.
// The per-frame color.NRGBA toolkit lives in the chroma/nrgba subpackage.
//
// The topology is a star: every space converts to and from exactly one hub,
// never directly to another leaf.
//
//	HSL, HSV  <->  SRGB  <->  RGB  <->  XYZ  <->  Lab  <->  LCh
//	                       Oklab <-'      `-> Luv
//	                         ^
//	                       OkLCh
//
// RGB is linear light in sRGB primaries; SRGB is the gamma-encoded form.
// XYZ uses the D65 white point (see D65). Hue is in turns [0, 1) — a fraction of the full circle — and is
// 0 for achromatic colors. Lab and Luv use L in 0..100; Oklab uses L in 0..1.
//
// Conversions never clamp: out-of-gamut input produces out-of-range output
// and round-trips exactly. Use SRGB.Clamp when you need 0..1 values.
package chroma
