// Package l0 implements L0-regularized image downscaling, in the spirit of
// Liu, He, Lau, Heng 2018, https://doi.org/10.1109/TIP.2017.2772838.
//
// The output is optimized so that its upsampled reconstruction matches the
// source while its gradients are sparse, which keeps strong edges and drops
// texture. This is a reconstruction from the L0 framework, not a port of the
// reference code; it uses a gradient-descent inner loop rather than the
// paper's FFT solve, so large outputs are slow.
//
//	out, err := l0.Resize(pix.FromImage(src), w, h, l0.DefaultLambda)
//
// Larger lambda removes more texture.
package l0
