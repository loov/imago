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
// XYZ uses the D65 white point. Hue is in turns [0, 1), a fraction of the full circle, and is
// 0 for achromatic colors. Lab and Luv use L in 0..100; Oklab uses L in 0..1.
//
// Conversions never clamp: out-of-gamut input produces out-of-range output.
// Round-trips are exact (to floating-point error) wherever the conversion is
// invertible. Singular cases lose information: HSL/HSV drop hue when S == 0
// or at L == 0 or 1 (HSL) / V == 0 (HSV); LCh and OkLCh drop hue when C == 0;
// Luv maps black to L = U = V = 0 and back, but u'v' are undefined at Y == 0.
// Use SRGB.Clamp when you need 0..1 values.
package chroma
