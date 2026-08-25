// Package l0 implements L0-regularized image downscaling.
package l0

import (
	"errors"
	"math"

	"github.com/loov/imago/pix"
)

// DefaultLambda is a gradient-sparsity weight that keeps strong edges
// while flattening low-amplitude texture; values in [0.005, 0.05] are sensible.
const DefaultLambda = 0.01

// Resize downsizes src following Liu, He, Lau and Heng,
// "L0-Regularized Image Downscaling" (IEEE TIP 2018).
// See https://doi.org/10.1109/TIP.2017.2772838.
//
// The output D minimizes ‖U(D) − I‖² + λ‖∇D‖₀ where U is bilinear upsampling
// to the input size, solved by half-quadratic splitting: gradients of D whose
// squared magnitude is below λ/β are pushed to zero, and the remaining
// least-squares problem is solved by gradient descent while β grows.
// lambda controls sparsity; lambda == 0 yields the plain least-squares (box-like) result.
// The β loop exits early once the gradients of D match the auxiliary variables
// to within 1e-6, at which point growing β no longer changes the solution.
// Width and height must be positive and no larger than src in either dimension.
func Resize(src *pix.Image, width, height int, lambda float64) (*pix.Image, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("l0: empty image")
	}
	inputWidth, inputHeight := src.W, src.H
	if lambda < 0 || math.IsNaN(lambda) || math.IsInf(lambda, 0) {
		return nil, errors.New("l0: lambda must be finite and non-negative")
	}
	if width <= 0 || height <= 0 || width > inputWidth || height > inputHeight {
		return nil, errors.New("l0: output dimensions must be positive and no larger than the input")
	}
	if width == inputWidth && height == inputHeight {
		return src.Clone(), nil
	}

	out := pix.New(width, height)
	up := newUpsampler(inputWidth, inputHeight, width, height)
	for i := range 4 {
		out.SetChannel(i, solve(src.Channel(i), up, width, height, lambda))
	}

	return out.Clamp(), nil
}

// iterations counts outer β iterations across all channels of the last Resize (test hook).
var iterations int

// upsampler holds the bilinear weights mapping each input pixel to 4 output pixels.
type upsampler struct {
	index  [][4]int
	weight [][4]float64
	norm   []float64 // column sums of U, for normalizing Uᵀ
}

func newUpsampler(inputWidth, inputHeight, width, height int) *upsampler {
	up := &upsampler{
		index:  make([][4]int, inputWidth*inputHeight),
		weight: make([][4]float64, inputWidth*inputHeight),
		norm:   make([]float64, width*height),
	}
	for y := range inputHeight {
		sy := min(max((float64(y)+0.5)*float64(height)/float64(inputHeight)-0.5, 0), float64(height-1))
		y0 := int(sy)
		y1 := min(y0+1, height-1)
		fy := sy - float64(y0)
		for x := range inputWidth {
			sx := min(max((float64(x)+0.5)*float64(width)/float64(inputWidth)-0.5, 0), float64(width-1))
			x0 := int(sx)
			x1 := min(x0+1, width-1)
			fx := sx - float64(x0)
			i := x + y*inputWidth
			up.index[i] = [4]int{x0 + y0*width, x1 + y0*width, x0 + y1*width, x1 + y1*width}
			up.weight[i] = [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}
			for k := range 4 {
				up.norm[up.index[i][k]] += up.weight[i][k]
			}
		}
	}
	return up
}

func solve(input []float64, up *upsampler, width, height int, lambda float64) []float64 {
	n := width * height
	// Initialize with the normalized least-squares transpose (an area average).
	d := make([]float64, n)
	for i := range input {
		for k := range 4 {
			d[up.index[i][k]] += up.weight[i][k] * input[i]
		}
	}
	for i := range d {
		d[i] /= up.norm[i]
	}

	h, v, grad := make([]float64, n), make([]float64, n), make([]float64, n)
	// ponytail: fixed β schedule and gradient-descent inner loop instead of the
	// paper's FFT solve; converges slowly for large outputs, switch to FFT if too slow.
	beta := max(2*lambda, 1e-3)
	iterations = 0
	for range 32 {
		iterations++
		// Auxiliary gradient step: keep gradients with ‖(h,v)‖² ≥ λ/β, zero the rest.
		for y := range height {
			for x := range width {
				i := x + y*width
				var hh, vv float64
				if x+1 < width {
					hh = d[i+1] - d[i]
				}
				if y+1 < height {
					vv = d[i+width] - d[i]
				}
				if hh*hh+vv*vv < lambda/beta {
					hh, vv = 0, 0
				}
				h[i], v[i] = hh, vv
			}
		}
		// Least-squares step: gradient descent on ‖U(D) − I‖² + β(‖∂xD − h‖² + ‖∂yD − v‖²).
		step := 1 / (1 + 8*beta)
		for range 64 {
			clear(grad)
			for i := range input {
				var r float64
				for k := range 4 {
					r += up.weight[i][k] * d[up.index[i][k]]
				}
				r -= input[i]
				for k := range 4 {
					grad[up.index[i][k]] += up.weight[i][k] * r
				}
			}
			for y := range height {
				for x := range width {
					i := x + y*width
					g := grad[i] / up.norm[i]
					if x+1 < width {
						g -= beta * (d[i+1] - d[i] - h[i])
					}
					if x > 0 {
						g += beta * (d[i] - d[i-1] - h[i-1])
					}
					if y+1 < height {
						g -= beta * (d[i+width] - d[i] - v[i])
					}
					if y > 0 {
						g += beta * (d[i] - d[i-width] - v[i-width])
					}
					grad[i] = g
				}
			}
			for i := range d {
				d[i] -= step * grad[i]
			}
		}
		// Converged once ∂D matches (h,v): the penalty is zero, so growing β changes nothing.
		var residual float64
		for y := range height {
			for x := range width {
				i := x + y*width
				if x+1 < width {
					residual = max(residual, math.Abs(d[i+1]-d[i]-h[i]))
				}
				if y+1 < height {
					residual = max(residual, math.Abs(d[i+width]-d[i]-v[i]))
				}
			}
		}
		if residual < 1e-6 {
			break
		}
		beta *= 2
		if beta > 1e5 {
			break
		}
	}
	return d
}
