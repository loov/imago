// Package dither maps images onto a palette with ordered (Bayer) dithering.
//
// Error-diffusion dithering is already in the standard library as
// draw.FloydSteinberg. Ordered dithering trades its smoother tones for a
// fixed, repeatable pattern that survives further scaling and looks right
// on pixel art.
//
//	dst := dither.Ordered(src, palette, 4, 0.5)
//
// The matrix size is a power of two (2, 4, 8, ...); larger matrices give more
// tone levels. strength scales the threshold offset; 1 is the classic
// full-strength pattern, 0 is plain nearest-color.
package dither
