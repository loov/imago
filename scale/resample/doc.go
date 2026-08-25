// Package resample implements separable filtered image resampling, up or
// down.
//
//	out, err := resample.Resize(pix.FromImage(src), w, h, resample.Lanczos3)
//
// Lanczos3 is sharpest and may ring; CatmullRom is a good general choice;
// MitchellNetravali is softer with no ringing. Mitchell(b, c) builds any
// filter from the Mitchell-Netravali family. Filters are scaled with the
// ratio when downscaling, so there is no aliasing at large reductions.
package resample
