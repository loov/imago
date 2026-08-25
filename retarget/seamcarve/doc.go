// Package seamcarve implements content-aware image shrinking by seam removal
// (Avidan, Shamir 2007, https://doi.org/10.1145/1275808.1276390).
//
// Resize removes low-energy vertical and horizontal seams until the image
// reaches the requested size. Only shrinking is supported; asking for a
// larger dimension returns an error.
//
//	out, err := seamcarve.Resize(src, src.Bounds().Dx()*3/4, src.Bounds().Dy())
//
// Energy is the gradient of premultiplied color plus alpha, so transparent
// regions carve first. Cost is O(w*h) per removed seam.
package seamcarve
