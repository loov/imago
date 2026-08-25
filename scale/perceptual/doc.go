// Package perceptual implements SSIM-based perceptual image downscaling
// (Öztireli, Gross 2015, https://doi.org/10.1145/2766891).
//
// The output is a box downscale adjusted so its local variance matches the
// source, which restores the contrast that averaging removes. Results look
// sharper than box at the same size without ringing.
//
//	out, err := perceptual.Resize(pix.FromImage(src), w, h)
//
// Works per channel on whatever encoding the input has.
package perceptual
