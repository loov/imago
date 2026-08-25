// Package ssim implements the SSIM (Wang et al. 2004) and MS-SSIM (Wang,
// Simoncelli, Bovik 2003) image similarity indices.
//
// Both indices compare images on luma only: Rec. 601 Y computed from the
// sRGB-encoded straight (un-premultiplied) RGB channels in [0, 1]. Alpha is
// otherwise ignored.
//
//	score, err := ssim.SSIM(pix.FromImage(a), pix.FromImage(b))
//
// Both images must have the same size. Scores are in [-1, 1], with 1 for
// identical images. SSIM uses an 11x11 Gaussian window; MSSSIM averages
// five dyadic scales and needs images at least 176 pixels on each side.
package ssim
