// Package quant builds color palettes from images.
//
// MedianCut and Wu generate a palette from scratch; KMeans refines any
// palette against an image in Oklab. Wu gives better palettes than median
// cut at similar cost; k-means on top of Wu is the best this package does.
//
//	p := quant.KMeans(src, quant.Wu(src, 16), 8)
//
//	dst := image.NewPaletted(src.Bounds(), p)
//	draw.FloydSteinberg.Draw(dst, dst.Rect, src, src.Bounds().Min)
//
// Input is *image.NRGBA like the other 8-bit packages. All functions look at
// straight RGB and ignore alpha; fully transparent pixels are skipped and
// palette entries are opaque. Use image/draw or package dither to apply a
// palette.
package quant
