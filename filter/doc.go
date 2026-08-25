// Package filter implements fast approximations of repeated 3x3 filters
// over a byte plane.
//
// A Channel is one 8-bit plane (a mask, alpha, or one color channel). Unlike
// the rest of the module, filters mutate the Channel in place; use Clone
// first to keep the original.
//
//	ch := filter.NewChannel(w, h)
//	copy(ch.Data, gray)
//	base := ch.Clone()
//	base.Erode(4) // dark lines up to ~4px wide disappear
//	base.Blur(4)  // roughly a Gaussian with the variance of four 3x3 box blurs
//
// steps means "equivalent to this many 3x3 passes". Blur and Erode run in
// O(N) regardless of steps; Median really does run steps passes and is meant
// for small values. Erode is a max filter: dark features shrink, bright ones
// grow. Blur(1) is a no-op; see the method docs for the exact kernels.
package filter
