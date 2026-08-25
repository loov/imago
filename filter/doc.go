// Package filter implements separable 3-tap filters over a byte plane.
//
// A Channel is one 8-bit plane (a mask, alpha, or one color channel). Unlike
// the rest of the module, filters mutate the Channel in place; use Clone
// first to keep the original.
//
//	ch := filter.NewChannel(w, h)
//	copy(ch.Data, mask)
//	soft := ch.Clone()
//	soft.Erode(1) // shrink the mask by one pixel
//	soft.Blur(3)  // three passes of a 3-tap box blur
//
// Each call runs steps passes of a 3x3 kernel, so larger radii come from
// repeating the filter rather than a larger kernel.
package filter
