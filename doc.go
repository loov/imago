// Package imago is the root of a stdlib-only collection of image algorithms
// and color math for Go. It contains no code; see the subpackages.
//
// Float algorithms (scale/*, metric/ssim) work on *pix.Image, premultiplied
// float RGBA. Exact 8-bit algorithms (pixelart, retarget/seamcarve,
// enhance/sketch) work on *image.NRGBA. Color math lives in chroma (float64
// color spaces) and chroma/nrgba (per-frame color.NRGBA helpers).
package imago
