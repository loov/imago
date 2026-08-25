// Package contentadaptive implements content-adaptive image downscaling
// (Kopf, Shamir, Peers 2013, https://doi.org/10.1145/2508363.2508370).
//
// The algorithm fits one anisotropic Gaussian kernel per output pixel in
// CIELAB and iterates so that each kernel covers a region of similar color.
// Small features and thin lines survive at ratios where box averaging would
// erase them.
//
//	out, err := contentadaptive.Resize(pix.FromImage(src), w, h)
//
// Unlike the other scalers this one interprets its input as sRGB-encoded,
// because the Lab conversion needs to know the encoding. Do not Linearize
// first. It is the slowest scaler in the module: about 40 s for a 1024x1024
// source on a 2024 laptop, mostly independent of the target size.
package contentadaptive
