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

// upsampler holds separable bilinear tables mapping input pixel (x, y) to the
// four output pixels (x0|x1, y0|y1) with weights (1-wx|wx)·(1-wy|wy).
// Storage is O(inputWidth + inputHeight) instead of per-input-pixel taps.
type upsampler struct {
	x0, x1 []int
	wx     []float64
	y0, y1 []int
	wy     []float64
	norm   []float64 // column sums of U (= Σ over input pixels of tap weight), for normalizing Uᵀ
}

// axis fills the tables for one axis; input pixel centers map to output coordinates.
func axis(input, output int) (i0, i1 []int, w []float64, colSum []float64) {
	i0, i1, w, colSum = make([]int, input), make([]int, input), make([]float64, input), make([]float64, output)
	for i := range input {
		s := min(max((float64(i)+0.5)*float64(output)/float64(input)-0.5, 0), float64(output-1))
		i0[i] = int(s)
		i1[i] = min(i0[i]+1, output-1)
		w[i] = s - float64(i0[i])
		colSum[i0[i]] += 1 - w[i]
		colSum[i1[i]] += w[i]
	}
	return
}

func newUpsampler(inputWidth, inputHeight, width, height int) *upsampler {
	up := &upsampler{norm: make([]float64, width*height)}
	var nx, ny []float64
	up.x0, up.x1, up.wx, nx = axis(inputWidth, width)
	up.y0, up.y1, up.wy, ny = axis(inputHeight, height)
	// U's column sum is separable: Σ_x Σ_y wx·wy = (Σ_x wx)(Σ_y wy).
	for y := range height {
		for x := range width {
			up.norm[x+y*width] = nx[x] * ny[y]
		}
	}
	return up
}

// residualT accumulates Uᵀ(U·d − input) into grad; with d == nil it accumulates −Uᵀ·input.
// The four taps of input pixel (x, y) are (x0|x1, y0|y1) with weights (1-wx|wx)·(1-wy|wy).
func (up *upsampler) residualT(d, input, grad []float64, width int) {
	inputWidth := len(up.x0)
	for y := range up.y0 {
		r0, r1, fy := up.y0[y]*width, up.y1[y]*width, up.wy[y]
		row := input[y*inputWidth : (y+1)*inputWidth]
		for x, in := range row {
			fx := up.wx[x]
			i00, i10, i01, i11 := up.x0[x]+r0, up.x1[x]+r0, up.x0[x]+r1, up.x1[x]+r1
			w00, w10, w01, w11 := (1-fx)*(1-fy), fx*(1-fy), (1-fx)*fy, fx*fy
			r := -in
			if d != nil {
				r += w00*d[i00] + w10*d[i10] + w01*d[i01] + w11*d[i11]
			}
			grad[i00] += w00 * r
			grad[i10] += w10 * r
			grad[i01] += w01 * r
			grad[i11] += w11 * r
		}
	}
}

func solve(input []float64, up *upsampler, width, height int, lambda float64) []float64 {
	n := width * height
	// Initialize with the normalized least-squares transpose (an area average).
	d := make([]float64, n)
	up.residualT(nil, input, d, width)
	for i := range d {
		d[i] = -d[i] / up.norm[i]
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
		// Least-squares step: preconditioned gradient descent on
		// f(D) = ½‖U(D) − I‖² + ½β(‖∂xD − h‖² + ‖∂yD − v‖²), with gradient
		// g = Uᵀ(UD − I) + β∇ᵀ(∇D − (h,v)) and Jacobi preconditioner
		// P_i = norm_i + 2β·deg_i (deg = number of grid neighbours of i).
		// Stability: the Hessian A = UᵀU + β∇ᵀ∇ has absolute row sums
		// Σ_j|A_ij| ≤ norm_i + 2β·deg_i = P_i (bilinear weights of each input
		// pixel sum to 1, so row i of UᵀU sums to norm_i; row i of ∇ᵀ∇ has
		// diagonal deg_i and deg_i entries of −1). By Gershgorin the eigenvalues
		// of P⁻¹A lie in [0, 1], so D ← D − P⁻¹g (step 1) never diverges.
		for range 64 {
			clear(grad)
			up.residualT(d, input, grad, width)
			for y := range height {
				for x := range width {
					i := x + y*width
					g := grad[i]
					deg := 0
					if x+1 < width {
						g -= beta * (d[i+1] - d[i] - h[i])
						deg++
					}
					if x > 0 {
						g += beta * (d[i] - d[i-1] - h[i-1])
						deg++
					}
					if y+1 < height {
						g -= beta * (d[i+width] - d[i] - v[i])
						deg++
					}
					if y > 0 {
						g += beta * (d[i] - d[i-width] - v[i-width])
						deg++
					}
					grad[i] = g / (up.norm[i] + 2*beta*float64(deg))
				}
			}
			for i := range d {
				d[i] -= grad[i]
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
