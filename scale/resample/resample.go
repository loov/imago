// Package resample implements separable filtered image resampling.
package resample

import (
	"errors"
	"math"

	"github.com/loov/imago/pix"
)

// Filter is a symmetric reconstruction kernel; Weight is evaluated on
// [-Support, Support].
type Filter struct {
	Support float64
	Weight  func(x float64) float64
}

var (
	// Lanczos3 is the sinc-windowed sinc kernel with three lobes.
	Lanczos3 = Filter{Support: 3, Weight: func(x float64) float64 {
		x = math.Abs(x)
		if x == 0 {
			return 1
		}
		if x >= 3 {
			return 0
		}
		px := math.Pi * x
		return 3 * math.Sin(px) * math.Sin(px/3) / (px * px)
	}}
	// CatmullRom is the Keys cubic (a = -0.5), also called bicubic.
	CatmullRom = Mitchell(0, 0.5)
	// MitchellNetravali is the recommended B = C = 1/3 cubic.
	MitchellNetravali = Mitchell(1.0/3, 1.0/3)
)

// Mitchell returns the cubic from Mitchell and Netravali, "Reconstruction
// Filters in Computer Graphics" (1988), https://doi.org/10.1145/378456.378514.
// B = 1, C = 0 is the cubic B-spline; B = 0, C = 0.5 is Catmull-Rom.
func Mitchell(b, c float64) Filter {
	return Filter{Support: 2, Weight: func(x float64) float64 {
		x = math.Abs(x)
		if x < 1 {
			return ((12-9*b-6*c)*x*x*x + (-18+12*b+6*c)*x*x + (6 - 2*b)) / 6
		}
		if x < 2 {
			return ((-b-6*c)*x*x*x + (6*b+30*c)*x*x + (-12*b-48*c)*x + (8*b + 24*c)) / 6
		}
		return 0
	}}
}

// Resize scales src to width x height using a two-pass separable filter.
// When downscaling the kernel is widened by the scale ratio so it acts as a
// low-pass filter. Width and height must be positive. Filtering happens in
// the encoding of src; wrap with (*pix.Image).Linearize and Delinearize to
// resample in linear light.
func Resize(src *pix.Image, width, height int, f Filter) (*pix.Image, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("resample: empty image")
	}
	inputWidth, inputHeight := src.W, src.H
	if width <= 0 || height <= 0 {
		return nil, errors.New("resample: output dimensions must be positive")
	}
	if width == inputWidth && height == inputHeight {
		return src.Clone(), nil
	}

	out := pix.New(width, height)
	out.Pix = pass(f, src.Pix, inputWidth, inputHeight, width, 4, 4*inputWidth, 4, 4*width)
	out.Pix = pass(f, out.Pix, inputHeight, width, height, 4*width, 4, 4*width, 4)
	return out, nil
}

// pass resamples one axis from n to m samples across the given number of
// lines. Sample j of a line is at j*stride + line*lineStride in src and
// i*dstStride + line*dstLineStride in the result.
func pass(f Filter, src []float64, n, lines, m, stride, lineStride, dstStride, dstLineStride int) []float64 {
	scale := float64(n) / float64(m)
	widen := max(scale, 1)
	support := f.Support * widen
	dst := make([]float64, 4*m*lines)
	for i := range m {
		center := (float64(i)+0.5)*scale - 0.5
		lo := int(math.Ceil(center - support))
		hi := int(math.Floor(center + support))
		var weights []float64
		var sum float64
		for j := lo; j <= hi; j++ {
			w := f.Weight((float64(j) - center) / widen)
			weights = append(weights, w)
			sum += w
		}
		for line := range lines {
			var acc [4]float64
			for k, w := range weights {
				o := min(max(lo+k, 0), n-1)*stride + line*lineStride
				for c := range 4 {
					acc[c] += w * src[o+c]
				}
			}
			o := i*dstStride + line*dstLineStride
			for c := range 4 {
				dst[o+c] = acc[c] / sum
			}
		}
	}
	return dst
}
