// Package box implements area-average image downscaling.
//
// Every output pixel is the exact area-weighted mean of the source pixels it
// covers, including fractional coverage at the edges. It is the baseline the
// other scalers are measured against and the right default when you have no
// reason to prefer another.
//
//	out, err := box.Resize(pix.FromImage(src), w, h)
//
// Upscaling is not supported. For linear-light averaging see pix.Linearize.
package box
