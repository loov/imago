// Package dpid implements detail-preserving image downscaling.
package dpid

import (
	"errors"
	"math"

	"github.com/loov/imago/pix"
)

// Resize downsizes src using the guided weighting from Weber, Waechter,
// Amende, Magnor and Goesele, "Rapid, Detail-Preserving Image Downscaling"
// (SIGGRAPH Asia 2016). See https://doi.org/10.1145/2980179.2980239.
//
// Each output pixel is a weighted average of its source region, where a
// source pixel's weight is (‖I(p) − Ĩ(q)‖/2)^lambda, the distance taken over
// the four premultiplied channels (R, G, B, A) so an opacity-only edge counts
// as detail. Ĩ is the box-downscaled image smoothed with the paper's kernel
// [1 2 1; 2 4 2; 1 2 1]/16 (Eq. 1); at borders the out-of-range taps are
// dropped and the remaining weights renormalized. Pixels differing from the
// local average contribute more. lambda=0 reduces exactly to box averaging,
// the paper's default is 1.0. Width and height must be positive and no larger
// than src in either dimension.
func Resize(src *pix.Image, width, height int, lambda float64) (*pix.Image, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("dpid: empty image")
	}
	inputWidth, inputHeight := src.W, src.H
	if width <= 0 || height <= 0 || width > inputWidth || height > inputHeight {
		return nil, errors.New("dpid: output dimensions must be positive and no larger than the input")
	}
	if lambda < 0 || math.IsNaN(lambda) {
		return nil, errors.New("dpid: lambda must be non-negative")
	}
	if width == inputWidth && height == inputHeight {
		return src.Clone(), nil
	}

	// Guidance: box downscale with fractional coverage.
	box := make([][4]float64, width*height)
	for y := range height {
		for x := range width {
			var sum [4]float64
			var total float64
			forRegion(x, y, width, height, inputWidth, inputHeight, func(p [4]float64, cover float64) {
				for i := range sum {
					sum[i] += cover * p[i]
				}
				total += cover
			}, src)
			for i := range sum {
				sum[i] /= total
			}
			box[x+y*width] = sum
		}
	}

	out := pix.New(width, height)
	for y := range height {
		for x := range width {
			guide := smoothGuide(box, x, y, width, height)

			var sum [4]float64
			var total float64
			forRegion(x, y, width, height, inputWidth, inputHeight, func(p [4]float64, cover float64) {
				var d2 float64
				for i := range p {
					d := p[i] - guide[i]
					d2 += d * d
				}
				w := cover * math.Pow(math.Sqrt(d2/4), lambda)
				for i := range sum {
					sum[i] += w * p[i]
				}
				total += w
			}, src)
			if total == 0 {
				sum = box[x+y*width]
				total = 1
			}

			out.Set(x, y, sum[0]/total, sum[1]/total, sum[2]/total, sum[3]/total)
		}
	}
	return out, nil
}

// smoothGuide smooths box at (x, y) with the kernel [1 2 1; 2 4 2; 1 2 1]/16,
// omitting out-of-range taps and renormalizing the remaining weights.
func smoothGuide(box [][4]float64, x, y, width, height int) [4]float64 {
	var guide [4]float64
	var total float64
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			xx, yy := x+dx, y+dy
			if xx < 0 || yy < 0 || xx >= width || yy >= height {
				continue
			}
			w := float64((2 - dx*dx) * (2 - dy*dy))
			g := box[xx+yy*width]
			for i := range guide {
				guide[i] += w * g[i]
			}
			total += w
		}
	}
	for i := range guide {
		guide[i] /= total
	}
	return guide
}

// forRegion calls fn for every input pixel overlapping output pixel (x, y)
// with its fractional coverage.
func forRegion(x, y, width, height, inputWidth, inputHeight int, fn func(p [4]float64, cover float64), input *pix.Image) {
	x0, x1 := float64(x)*float64(inputWidth)/float64(width), float64(x+1)*float64(inputWidth)/float64(width)
	y0, y1 := float64(y)*float64(inputHeight)/float64(height), float64(y+1)*float64(inputHeight)/float64(height)
	for yy := int(y0); yy < min(int(math.Ceil(y1)), inputHeight); yy++ {
		cy := min(y1, float64(yy+1)) - max(y0, float64(yy))
		for xx := int(x0); xx < min(int(math.Ceil(x1)), inputWidth); xx++ {
			cx := min(x1, float64(xx+1)) - max(x0, float64(xx))
			if cx > 0 && cy > 0 {
				i := 4 * (xx + yy*inputWidth)
				fn([4]float64(input.Pix[i:i+4]), cx*cy)
			}
		}
	}
}
