// Package pixelate downscales an image and builds its palette at the same
// time, after Gerstner, DeCarlo, Alexa, Finkelstein, Gingold, Nealen,
// "Pixelated Image Abstraction" (NPAR 2012,
// https://doi.org/10.2312/PE/NPAR/NPAR12/029-036).
//
// Each output pixel is a SLIC superpixel of the source; superpixel colors
// and a palette of k colors are optimized together with deterministic
// annealing in CIELAB, so the palette is chosen for what survives at the
// target size rather than for the source.
//
//	dst, err := pixelate.Resize(pix.FromImage(src), 64, 64, 16)
//
// Like scale/contentadaptive this reads src as sRGB-encoded (do not
// Linearize first). Output is an *image.Paletted with an opaque palette;
// alpha is ignored. Cost is around 1.5 s per source megapixel.
package pixelate
