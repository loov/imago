// Package dpid implements detail-preserving image downscaling (Weber et al.
// 2016, https://doi.org/10.1145/2980179.2980239).
//
// Pixels that differ from the local box average get more weight, so edges
// and small details stay visible. lambda controls how strongly; 0 is a plain
// box filter, 1 matches the paper's default, larger values look sharper and
// noisier.
//
//	out, err := dpid.Resize(pix.FromImage(src), w, h, 1)
//
// Fast enough for interactive use; a good first choice over box for
// thumbnails.
package dpid
