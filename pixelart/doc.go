// Package pixelart implements pixel-art upscalers: Scale2x, Scale3x and xBR.
//
// These are exact integer-ratio upscalers that keep hard edges instead of
// blurring, meant for sprites and low-resolution UI:
//
//	big := pixelart.XBR2x(sprite) // 2x, smooths diagonals
//	big = pixelart.Scale3x(sprite) // 3x, pure edge replication
//
// Inputs are *image.NRGBA and pixels are compared exactly; dithered or
// photographic input gets no benefit over resample.
package pixelart
