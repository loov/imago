package contentadaptive

import (
	"errors"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/loov/imago/chroma"
	"github.com/loov/imago/pix"
)

const (
	// Deviates from pseudocode line 7 ("loop until no change"): a fixed cap and
	// an epsilon bound the paper's acknowledged non-convergent cases.
	maxIterations      = 64
	convergenceEpsilon = 1e-6
	// Spatial variance limits on the singular values of Σ in normalized
	// (output-pixel) coordinates.
	minSpatialVariance = 0.05
	maxSpatialVariance = 0.1
	// Initial color standard deviation σ; grows by 10% per constraint violation.
	// The pseudocode gives no Lab scale for σ=1e-4; we assume Lab normalized to ~0..1.
	initialColorSigma = 1e-4
)

// Lab normalized to roughly 0..1, the scale initialColorSigma assumes.
type lab struct {
	l, a, b float64
}

func normalize(c chroma.Lab) lab {
	return lab{c.L / 100, (c.A + 86.185) / 184.419, (c.B + 107.863) / 202.345}
}

func (c lab) denormalize() chroma.Lab {
	return chroma.Lab{L: c.l * 100, A: c.a*184.419 - 86.185, B: c.b*202.345 - 107.863}
}

type pixel struct {
	color lab
	alpha float64
}

type kernel struct {
	gridX, gridY           int
	centerX, centerY       float64
	meanX, meanY           float64
	covXX, covXY, covYY    float64
	color                  lab
	alpha, sigma           float64
	x0, y0, x1, y1, stride int
	gamma                  []float64 // window into the shared per-Resize buffer
}

// state is the subset of kernel that convergence is measured on.
type state struct {
	meanX, meanY        float64
	covXX, covXY, covYY float64
	color               lab
	alpha, sigma        float64
}

func (k *kernel) state() state {
	return state{k.meanX, k.meanY, k.covXX, k.covXY, k.covYY, k.color, k.alpha, k.sigma}
}

// Resize downsizes src using the constrained bilateral-kernel method from
// Kopf, Shamir, and Peers, "Content-Adaptive Image Downscaling" (2013).
// See https://doi.org/10.1145/2508363.2508370.
// Width and height must be positive and no larger than src in either dimension.
//
// Spatial quantities (pixel positions, kernel means, covariances) are in input
// pixel units; the covariance clamp is applied in normalized output-pixel units.
//
// Unlike the other scale packages, Resize interprets src as sRGB-encoded: it
// decodes to CIELAB, runs there, and re-encodes sRGB on output. Do not wrap it
// in Linearize/Delinearize. A same-size call returns a clone without the Lab
// round trip.
func Resize(src *pix.Image, width, height int) (*pix.Image, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("contentadaptive: empty image")
	}
	inputWidth, inputHeight := src.W, src.H
	if width <= 0 || height <= 0 || width > inputWidth || height > inputHeight {
		return nil, errors.New("contentadaptive: output dimensions must be positive and no larger than the input")
	}
	if width == inputWidth && height == inputHeight {
		return src.Clone(), nil
	}

	pixels := readPixels(src)
	rx := float64(inputWidth) / float64(width)
	ry := float64(inputHeight) / float64(height)
	kernels := initializeKernels(inputWidth, inputHeight, width, height, rx, ry)
	pixelSum := make([]float64, len(pixels))
	previous := make([]state, len(kernels))
	neighborMeans := make([][2]float64, len(kernels))

	for range maxIterations {
		expectation(kernels, pixels, inputWidth, inputHeight, width, height, rx, ry, pixelSum)
		maximize(kernels, pixels, inputWidth, previous)
		correct(kernels, inputWidth, inputHeight, width, height, rx, ry, neighborMeans)
		if converged(previous, kernels) {
			break
		}
	}

	return render(kernels, width, height), nil
}

// readPixels converts premultiplied sRGB to straight normalized Lab, the
// algorithm's own color space.
func readPixels(m *pix.Image) []pixel {
	pixels := make([]pixel, m.W*m.H)
	for i := range pixels {
		r, g, b, a := m.Pix[4*i], m.Pix[4*i+1], m.Pix[4*i+2], m.Pix[4*i+3]
		if a > 0 {
			r, g, b = r/a, g/a, b/a
		}
		pixels[i] = pixel{color: normalize(chroma.LabFromXYZ(chroma.SRGB{R: r, G: g, B: b}.RGB().XYZ())), alpha: a}
	}
	return pixels
}

func initializeKernels(inputWidth, inputHeight, width, height int, rx, ry float64) []kernel {
	kernels := make([]kernel, width*height)
	// Supports are bounded by (4rx+1)(4ry+1), so one backing slice holds every gamma.
	gamma := make([]float64, 0, width*height*(int(4*rx)+1)*(int(4*ry)+1))
	for y := range height {
		for x := range width {
			// Deviates from pseudocode line 20/table: pixel centers are at x+½ so
			// they share the kernel centers' ½ offset (see expectation).
			centerX := (float64(x) + 0.5) * rx
			centerY := (float64(y) + 0.5) * ry
			x0 := max(0, int(math.Floor(centerX-2*rx))+1)
			y0 := max(0, int(math.Floor(centerY-2*ry))+1)
			x1 := min(inputWidth, int(math.Ceil(centerX+2*rx)))
			y1 := min(inputHeight, int(math.Ceil(centerY+2*ry)))
			stride := x1 - x0
			n := stride * (y1 - y0)
			gamma = gamma[:len(gamma)+n]
			kernels[x+y*width] = kernel{
				gridX: x, gridY: y,
				centerX: centerX, centerY: centerY,
				meanX: centerX, meanY: centerY,
				covXX: rx * rx / 9, covYY: ry * ry / 9,
				color: lab{l: 0.5, a: 0.5, b: 0.5}, alpha: 0.5, sigma: initialColorSigma,
				x0: x0, y0: y0, x1: x1, y1: y1, stride: stride,
				gamma: gamma[len(gamma)-n : len(gamma) : len(gamma)],
			}
		}
	}
	return kernels
}

// expectation computes γ: each kernel's weights are first normalized over its
// own support (pseudocode lines 23–26), then per pixel across kernels (line 30).
// Both passes work in the log domain since σ makes raw weights underflow: a
// pixel's weights can all underflow linearly and must still normalize to a
// sum of 1 across the kernels covering it. The per-pixel pass converts to
// linear in place, so the final pass is a plain division.
func expectation(kernels []kernel, pixels []pixel, inputWidth, inputHeight, width, height int, rx, ry float64, pixelSum []float64) {
	parallelFor(len(kernels), func(lo, hi int) {
		for ki := lo; ki < hi; ki++ {
			k := &kernels[ki]
			determinant := k.covXX*k.covYY - k.covXY*k.covXY
			inverseXX := k.covYY / determinant
			inverseXY := -k.covXY / determinant
			inverseYY := k.covXX / determinant
			twoSigmaSquared := 2 * k.sigma * k.sigma

			maxLogWeight := math.Inf(-1)
			for y := k.y0; y < k.y1; y++ {
				for x := k.x0; x < k.x1; x++ {
					p := pixels[x+y*inputWidth]
					dx, dy := float64(x)+0.5-k.meanX, float64(y)+0.5-k.meanY
					dl, da, db := p.color.l-k.color.l, p.color.a-k.color.a, p.color.b-k.color.b
					// Deviates from pseudocode line 21: color distance is scaled by
					// alpha so transparent pixels carry no color.
					logWeight := -0.5*(dx*dx*inverseXX+2*dx*dy*inverseXY+dy*dy*inverseYY) -
						p.alpha*(dl*dl+da*da+db*db)/twoSigmaSquared
					k.gamma[(x-k.x0)+(y-k.y0)*k.stride] = logWeight
					maxLogWeight = max(maxLogWeight, logWeight)
				}
			}
			var kernelSum float64
			for _, logWeight := range k.gamma {
				kernelSum += math.Exp(logWeight - maxLogWeight)
			}
			kernelNormalization := maxLogWeight + math.Log(kernelSum)
			for i := range k.gamma {
				k.gamma[i] -= kernelNormalization
			}
		}
	})

	// Per-pixel normalization, inverted to iterate over the kernels covering
	// each pixel: rows are independent, and the fixed kernel order keeps the
	// sums deterministic under any level of parallelism. Supports reach at
	// most 2r+1 input pixels from the kernel center, under 3.5 grid units.
	parallelFor(inputHeight, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			gy0 := max(0, int(float64(y)/ry-3.5))
			gy1 := min(height-1, int(float64(y)/ry+3.5))
			for x := 0; x < inputWidth; x++ {
				gx0 := max(0, int(float64(x)/rx-3.5))
				gx1 := min(width-1, int(float64(x)/rx+3.5))
				maxLog := math.Inf(-1)
				for gy := gy0; gy <= gy1; gy++ {
					for gx := gx0; gx <= gx1; gx++ {
						k := &kernels[gx+gy*width]
						if x < k.x0 || x >= k.x1 || y < k.y0 || y >= k.y1 {
							continue
						}
						maxLog = max(maxLog, k.gamma[(x-k.x0)+(y-k.y0)*k.stride])
					}
				}
				var sum float64
				for gy := gy0; gy <= gy1; gy++ {
					for gx := gx0; gx <= gx1; gx++ {
						k := &kernels[gx+gy*width]
						if x < k.x0 || x >= k.x1 || y < k.y0 || y >= k.y1 {
							continue
						}
						index := (x - k.x0) + (y-k.y0)*k.stride
						e := math.Exp(k.gamma[index] - maxLog)
						k.gamma[index] = e
						sum += e
					}
				}
				pixelSum[x+y*inputWidth] = sum
			}
		}
	})

	parallelFor(len(kernels), func(lo, hi int) {
		for ki := lo; ki < hi; ki++ {
			k := &kernels[ki]
			for y := k.y0; y < k.y1; y++ {
				for x := k.x0; x < k.x1; x++ {
					if sum := pixelSum[x+y*inputWidth]; sum > 0 {
						k.gamma[(x-k.x0)+(y-k.y0)*k.stride] /= sum
					}
				}
			}
		}
	})
}

// maximize updates each kernel's parameters, saving the pre-update state into
// previous so convergence can be measured after correct.
func maximize(kernels []kernel, pixels []pixel, inputWidth int, previous []state) {
	parallelFor(len(kernels), func(lo, hi int) {
		for ki := lo; ki < hi; ki++ {
			k := &kernels[ki]
			previous[ki] = k.state()
			var weight, meanX, meanY, xx, xy, yy, l, a, b, alpha, colorWeight float64
			for y := k.y0; y < k.y1; y++ {
				for x := k.x0; x < k.x1; x++ {
					gamma := k.gamma[(x-k.x0)+(y-k.y0)*k.stride]
					p := pixels[x+y*inputWidth]
					px, py := float64(x)+0.5, float64(y)+0.5
					// Pseudocode line 37: covariance is taken about the old μ.
					dx, dy := px-k.meanX, py-k.meanY
					weight += gamma
					meanX += gamma * px
					meanY += gamma * py
					xx += gamma * dx * dx
					xy += gamma * dx * dy
					yy += gamma * dy * dy
					alpha += gamma * p.alpha
					// Deviates from pseudocode line 39: ν is an alpha-weighted mean so
					// transparent pixels carry no color.
					cw := gamma * p.alpha
					colorWeight += cw
					l += cw * p.color.l
					a += cw * p.color.a
					b += cw * p.color.b
				}
			}
			if weight == 0 {
				continue
			}
			k.covXX, k.covXY, k.covYY = xx/weight, xy/weight, yy/weight
			k.meanX, k.meanY = meanX/weight, meanY/weight
			k.alpha = alpha / weight
			if colorWeight > 0 {
				k.color = lab{l: l / colorWeight, a: a / colorWeight, b: b / colorWeight}
			}
		}
	})
}
func correct(kernels []kernel, inputWidth, inputHeight, width, height int, rx, ry float64, neighborMeans [][2]float64) {
	parallelFor(len(kernels), func(lo, hi int) {
		for i := lo; i < hi; i++ {
			k := &kernels[i]
			var x, y float64
			count := 0
			for _, offset := range cardinalOffsets {
				nx, ny := k.gridX+offset[0], k.gridY+offset[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				n := &kernels[nx+ny*width]
				x, y, count = x+n.meanX, y+n.meanY, count+1
			}
			if count == 0 {
				neighborMeans[i] = [2]float64{k.meanX, k.meanY}
				continue
			}
			neighborMeans[i] = [2]float64{x / float64(count), y / float64(count)}
		}
	})

	parallelFor(len(kernels), func(lo, hi int) {
		for i := lo; i < hi; i++ {
			k := &kernels[i]
			k.meanX = min(max((k.meanX+neighborMeans[i][0])/2, k.centerX-rx/4), k.centerX+rx/4)
			k.meanY = min(max((k.meanY+neighborMeans[i][1])/2, k.centerY-ry/4), k.centerY+ry/4)
			clampCovariance(k, rx, ry)
		}
	})

	// The σ updates write into neighbors as well, so the parallel pass only
	// counts them and the multiplications are applied serially afterwards;
	// nothing in this pass reads σ, so deferring is equivalent.
	bumps := make([]int32, len(kernels))
	parallelFor(len(kernels), func(lo, hi int) {
		var scratch edgeScratch
		for ki := lo; ki < hi; ki++ {
			k := &kernels[ki]
			for _, offset := range neighborOffsets {
				nx, ny := k.gridX+offset[0], k.gridY+offset[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				n := &kernels[nx+ny*width]
				dx, dy := float64(offset[0]), float64(offset[1])
				var weight, directionalVariance, edgeStrength float64
				for y := k.y0; y < k.y1; y++ {
					for x := k.x0; x < k.x1; x++ {
						gamma := k.gamma[(x-k.x0)+(y-k.y0)*k.stride]
						// Deviates from the pseudocode C-step shape constraint (s ← Σ γ max(0,(pi−μk)ᵀd)² vs
						// 0.2·rx): taken literally in input pixels the raw sum is ~10×
						// the limit for every kernel, so σ grows without bound and the
						// result blurs. Use the mean squared projection in normalized
						// (output-pixel) units against 0.2 instead.
						projection := max(0, (float64(x)+0.5-k.meanX)/rx*dx+(float64(y)+0.5-k.meanY)/ry*dy)
						weight += gamma
						directionalVariance += gamma * projection * projection
						edgeStrength += gamma * n.gammaAt(x, y)
					}
				}
				if weight == 0 {
					continue
				}

				falseEdge := false
				if (offset[0] == 0 || offset[1] == 0) && edgeStrength < 0.08 {
					ox, oy := edgeOrientation(k, n, inputWidth, inputHeight, &scratch)
					magnitude := math.Hypot(ox, oy)
					if magnitude > 0 {
						cosine := math.Abs(dx*ox+dy*oy) / (math.Hypot(dx, dy) * magnitude)
						falseEdge = min(cosine, 1) < math.Cos(25*math.Pi/180)
					}
				}
				if directionalVariance/weight > 0.2 || falseEdge {
					atomic.AddInt32(&bumps[ki], 1)
					atomic.AddInt32(&bumps[nx+ny*width], 1)
				}
			}
		}
	})
	for i, n := range bumps {
		for range n {
			kernels[i].sigma *= 1.1
		}
	}
}

var cardinalOffsets = [...][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

var neighborOffsets = [...][2]int{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// clampCovariance clamps the singular values of Σ to [min,max]SpatialVariance
// in normalized coordinates: Σn = D⁻¹ Σ D⁻ᵀ with D = diag(rx, ry).
func clampCovariance(k *kernel, rx, ry float64) {
	xx, xy, yy := k.covXX/(rx*rx), k.covXY/(rx*ry), k.covYY/(ry*ry)
	middle := (xx + yy) / 2
	radius := math.Hypot((xx-yy)/2, xy)
	larger := min(max(middle+radius, minSpatialVariance), maxSpatialVariance)
	smaller := min(max(middle-radius, minSpatialVariance), maxSpatialVariance)
	angle := math.Atan2(2*xy, xx-yy) / 2
	c, s := math.Cos(angle), math.Sin(angle)
	k.covXX = (larger*c*c + smaller*s*s) * rx * rx
	k.covXY = (larger - smaller) * c * s * rx * ry
	k.covYY = (larger*s*s + smaller*c*c) * ry * ry
}

func (k *kernel) gammaAt(x, y int) float64 {
	if x < k.x0 || x >= k.x1 || y < k.y0 || y >= k.y1 {
		return 0
	}
	return k.gamma[(x-k.x0)+(y-k.y0)*k.stride]
}

// edgeScratch holds three reusable ratio rows for edgeOrientation, one per
// goroutine.
type edgeScratch struct {
	vals [3][]float64
	ok   [3][]bool
}

// edgeOrientation estimates the gradient of the ratio field kγ/(kγ+nγ) over
// k's support. Each ratio row is computed once into scratch and shared by the
// central differences of the neighboring rows.
func edgeOrientation(k, n *kernel, width, height int, s *edgeScratch) (float64, float64) {
	w := k.x1 - k.x0 + 2 // ratios for x in [k.x0-1, k.x1]
	for i := range 3 {
		if cap(s.vals[i]) < w {
			s.vals[i] = make([]float64, w)
			s.ok[i] = make([]bool, w)
		}
		s.vals[i] = s.vals[i][:w]
		s.ok[i] = s.ok[i][:w]
	}
	fill := func(vals []float64, ok []bool, y int) {
		y = min(max(y, 0), height-1)
		for i := range w {
			x := min(max(k.x0-1+i, 0), width-1)
			kg, ng := k.gammaAt(x, y), n.gammaAt(x, y)
			if kg+ng == 0 {
				ok[i] = false
				continue
			}
			vals[i], ok[i] = kg/(kg+ng), true
		}
	}
	prev, cur, next := 0, 1, 2
	fill(s.vals[prev], s.ok[prev], k.y0-1)
	fill(s.vals[cur], s.ok[cur], k.y0)

	var ox, oy float64
	for y := k.y0; y < k.y1; y++ {
		fill(s.vals[next], s.ok[next], y+1)
		for x := k.x0; x < k.x1; x++ {
			i := x - k.x0 + 1
			center, centerOK := s.vals[cur][i], s.ok[cur][i]
			left, leftOK := s.vals[cur][i-1], s.ok[cur][i-1]
			right, rightOK := s.vals[cur][i+1], s.ok[cur][i+1]
			switch {
			case leftOK && rightOK:
				ox += (right - left) / 2
			case centerOK && rightOK:
				ox += right - center
			case leftOK && centerOK:
				ox += center - left
			}
			top, topOK := s.vals[prev][i], s.ok[prev][i]
			bottom, bottomOK := s.vals[next][i], s.ok[next][i]
			switch {
			case topOK && bottomOK:
				oy += (bottom - top) / 2
			case centerOK && bottomOK:
				oy += bottom - center
			case topOK && centerOK:
				oy += center - top
			}
		}
		prev, cur, next = cur, next, prev
	}
	return ox, oy
}

// parallelFor runs fn over [0, n) split into one contiguous chunk per
// available CPU.
func parallelFor(n int, fn func(lo, hi int)) {
	workers := min(runtime.GOMAXPROCS(0), n)
	if workers <= 1 {
		fn(0, n)
		return
	}
	var wg sync.WaitGroup
	for w := range workers {
		lo, hi := n*w/workers, n*(w+1)/workers
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(lo, hi)
		}()
	}
	wg.Wait()
}

func converged(previous []state, kernels []kernel) bool {
	for i := range kernels {
		k, p := kernels[i].state(), previous[i]
		if math.Abs(k.meanX-p.meanX) > convergenceEpsilon ||
			math.Abs(k.meanY-p.meanY) > convergenceEpsilon ||
			math.Abs(k.covXX-p.covXX) > convergenceEpsilon ||
			math.Abs(k.covXY-p.covXY) > convergenceEpsilon ||
			math.Abs(k.covYY-p.covYY) > convergenceEpsilon ||
			math.Abs(k.color.l-p.color.l) > convergenceEpsilon ||
			math.Abs(k.color.a-p.color.a) > convergenceEpsilon ||
			math.Abs(k.color.b-p.color.b) > convergenceEpsilon ||
			math.Abs(k.alpha-p.alpha) > convergenceEpsilon ||
			math.Abs(k.sigma-p.sigma) > convergenceEpsilon {
			return false
		}
	}
	return true
}

func render(kernels []kernel, width, height int) *pix.Image {
	m := pix.New(width, height)
	for i, k := range kernels {
		c := chroma.SRGBFromRGB(chroma.RGBFromXYZ(k.color.denormalize().XYZ())).Clamp()
		a := min(max(k.alpha, 0), 1)
		m.Pix[4*i], m.Pix[4*i+1], m.Pix[4*i+2], m.Pix[4*i+3] = c.R*a, c.G*a, c.B*a, a
	}
	return m
}
