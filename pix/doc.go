// Package pix provides a premultiplied float RGBA image, the common pixel
// representation for the float algorithms in this module.
//
// FromImage premultiplies any image.Image into float64 planes; NRGBA
// un-premultiplies and quantizes back:
//
//	m := pix.FromImage(src)
//	m, err := dpid.Resize(m, w, h, 1)
//	out := m.NRGBA()
//
// Values keep whatever encoding the source had, so a FromImage of an sRGB
// PNG is sRGB-encoded floats. To filter in linear light wrap the algorithm
// with Linearize and Delinearize:
//
//	m = m.Linearize()
//	m, err = box.Resize(m, w, h)
//	out := m.Delinearize().NRGBA()
//
// Channel and SetChannel expose the planes for algorithms that work one
// channel at a time. Results are at origin (0,0); the source's Bounds().Min
// is honored on input.
package pix
