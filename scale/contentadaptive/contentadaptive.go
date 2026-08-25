// Package contentadaptive implements content-adaptive image downscaling.
package contentadaptive

import (
	"errors"
	"math"

	"github.com/loov/imago/chroma"
	"github.com/loov/imago/pix"
)

const (
	// ponytail: a fixed cap bounds the paper's acknowledged non-convergent cases;
	// expose tuning only if real workloads need it.
	maxIterations      = 64
	convergenceEpsilon = 1e-6
	// Spatial variance limits, in units of the squared downscale ratio.
	minSpatialVariance = 0.05
	maxSpatialVariance = 0.1
	// Initial color variance; grows by 10% per constraint violation.
	initialColorVariance = 1e-4
)

// Lab normalized to roughly 0..1 so the color-variance constants apply.
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
	gamma                  []float64
}

// Resize downsizes src using the constrained bilateral-kernel method from
// Kopf, Shamir, and Peers, "Content-Adaptive Image Downscaling" (2013).
// See https://doi.org/10.1145/2508363.2508370.
// Width and height must be positive and no larger than src in either dimension.
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
	pixelMax := make([]float64, len(pixels))
	pixelSum := make([]float64, len(pixels))

	for range maxIterations {
		previous := append([]kernel(nil), kernels...)
		expectation(kernels, pixels, inputWidth, pixelMax, pixelSum)
		maximize(kernels, pixels, inputWidth)
		correct(kernels, inputWidth, inputHeight, width, height, rx, ry)
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
	for y := range height {
		for x := range width {
			centerX := (float64(x) + 0.5) * rx
			centerY := (float64(y) + 0.5) * ry
			x0 := max(0, int(math.Floor(centerX-2*rx))+1)
			y0 := max(0, int(math.Floor(centerY-2*ry))+1)
			x1 := min(inputWidth, int(math.Ceil(centerX+2*rx)))
			y1 := min(inputHeight, int(math.Ceil(centerY+2*ry)))
			stride := x1 - x0
			kernels[x+y*width] = kernel{
				gridX: x, gridY: y,
				centerX: centerX, centerY: centerY,
				meanX: centerX, meanY: centerY,
				covXX: rx * rx / 9, covYY: ry * ry / 9,
				color: lab{l: 0.5, a: 0.5, b: 0.5}, alpha: 0.5, sigma: initialColorVariance,
				x0: x0, y0: y0, x1: x1, y1: y1, stride: stride,
				gamma: make([]float64, stride*(y1-y0)),
			}
		}
	}
	return kernels
}

func expectation(kernels []kernel, pixels []pixel, inputWidth int, pixelMax, pixelSum []float64) {
	for i := range pixelMax {
		pixelMax[i] = math.Inf(-1)
		pixelSum[i] = 0
	}

	for ki := range kernels {
		k := &kernels[ki]
		determinant := k.covXX*k.covYY - k.covXY*k.covXY
		inverseXX := k.covYY / determinant
		inverseXY := -k.covXY / determinant
		inverseYY := k.covXX / determinant

		for y := k.y0; y < k.y1; y++ {
			for x := k.x0; x < k.x1; x++ {
				p := pixels[x+y*inputWidth]
				dx, dy := float64(x)+0.5-k.meanX, float64(y)+0.5-k.meanY
				dl, da, db := p.color.l-k.color.l, p.color.a-k.color.a, p.color.b-k.color.b
				// Transparent pixels carry no color, so their color distance is scaled by alpha.
				logWeight := -0.5*(dx*dx*inverseXX+2*dx*dy*inverseXY+dy*dy*inverseYY) -
					p.alpha*(dl*dl+da*da+db*db)/(2*k.sigma)
				index := (x - k.x0) + (y-k.y0)*k.stride
				k.gamma[index] = logWeight
				pixelIndex := x + y*inputWidth
				if logWeight <= pixelMax[pixelIndex] {
					pixelSum[pixelIndex] += math.Exp(logWeight - pixelMax[pixelIndex])
					continue
				}
				pixelSum[pixelIndex] = pixelSum[pixelIndex]*math.Exp(pixelMax[pixelIndex]-logWeight) + 1
				pixelMax[pixelIndex] = logWeight
			}
		}
	}

	for ki := range kernels {
		k := &kernels[ki]
		for y := k.y0; y < k.y1; y++ {
			for x := k.x0; x < k.x1; x++ {
				index := (x - k.x0) + (y-k.y0)*k.stride
				pixelIndex := x + y*inputWidth
				pixelNormalization := pixelMax[pixelIndex] + math.Log(pixelSum[pixelIndex])
				k.gamma[index] = math.Exp(k.gamma[index] - pixelNormalization)
			}
		}
	}
}

func maximize(kernels []kernel, pixels []pixel, inputWidth int) {
	for ki := range kernels {
		k := &kernels[ki]
		var weight, meanX, meanY, xx, xy, yy, l, a, b, alpha, colorWeight float64
		for y := k.y0; y < k.y1; y++ {
			for x := k.x0; x < k.x1; x++ {
				gamma := k.gamma[(x-k.x0)+(y-k.y0)*k.stride]
				p := pixels[x+y*inputWidth]
				px, py := float64(x)+0.5, float64(y)+0.5
				weight += gamma
				meanX += gamma * px
				meanY += gamma * py
				xx += gamma * px * px
				xy += gamma * px * py
				yy += gamma * py * py
				alpha += gamma * p.alpha
				// Premultiplied average: color contributes in proportion to its alpha.
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
		k.meanX, k.meanY = meanX/weight, meanY/weight
		k.covXX = xx/weight - k.meanX*k.meanX
		k.covXY = xy/weight - k.meanX*k.meanY
		k.covYY = yy/weight - k.meanY*k.meanY
		k.alpha = alpha / weight
		if colorWeight > 0 {
			k.color = lab{l: l / colorWeight, a: a / colorWeight, b: b / colorWeight}
		}
	}
}

func correct(kernels []kernel, inputWidth, inputHeight, width, height int, rx, ry float64) {
	neighborMeans := make([][2]float64, len(kernels))
	for i, k := range kernels {
		var x, y float64
		count := 0
		for _, offset := range cardinalOffsets {
			nx, ny := k.gridX+offset[0], k.gridY+offset[1]
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			n := kernels[nx+ny*width]
			x, y, count = x+n.meanX, y+n.meanY, count+1
		}
		if count == 0 {
			neighborMeans[i] = [2]float64{k.meanX, k.meanY}
			continue
		}
		neighborMeans[i] = [2]float64{x / float64(count), y / float64(count)}
	}

	for i := range kernels {
		k := &kernels[i]
		k.meanX = min(max((k.meanX+neighborMeans[i][0])/2, k.centerX-rx/4), k.centerX+rx/4)
		k.meanY = min(max((k.meanY+neighborMeans[i][1])/2, k.centerY-ry/4), k.centerY+ry/4)
		clampCovariance(k, rx*ry)
	}

	for ki := range kernels {
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
					projection := max(0, (float64(x)+0.5-k.meanX)*dx+(float64(y)+0.5-k.meanY)*dy)
					weight += gamma
					directionalVariance += gamma * projection * projection
					edgeStrength += gamma * n.gammaAt(x, y)
				}
			}
			if weight == 0 {
				continue
			}

			falseEdge := false
			if (offset[0] == 0 || offset[1] == 0) && edgeStrength < 0.08*rx*ry {
				ox, oy := edgeOrientation(k, n, inputWidth, inputHeight)
				magnitude := math.Hypot(ox, oy)
				if magnitude > 0 {
					cosine := math.Abs(dx*ox+dy*oy) / (math.Hypot(dx, dy) * magnitude)
					falseEdge = min(cosine, 1) < math.Cos(25*math.Pi/180)
				}
			}
			// The published expression calls s a variance, so normalize its
			// weighted mass and compare it with a squared spatial distance.
			directionalLimit := 0.2 * (dx*dx*rx*rx + dy*dy*ry*ry)
			if directionalVariance/weight > directionalLimit || falseEdge {
				k.sigma *= 1.1
				n.sigma *= 1.1
			}
		}
	}
}

var cardinalOffsets = [...][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

var neighborOffsets = [...][2]int{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

func clampCovariance(k *kernel, scale float64) {
	middle := (k.covXX + k.covYY) / 2
	radius := math.Hypot((k.covXX-k.covYY)/2, k.covXY)
	larger := min(max(middle+radius, minSpatialVariance*scale), maxSpatialVariance*scale)
	smaller := min(max(middle-radius, minSpatialVariance*scale), maxSpatialVariance*scale)
	angle := math.Atan2(2*k.covXY, k.covXX-k.covYY) / 2
	c, s := math.Cos(angle), math.Sin(angle)
	k.covXX = larger*c*c + smaller*s*s
	k.covXY = (larger - smaller) * c * s
	k.covYY = larger*s*s + smaller*c*c
}

func (k *kernel) gammaAt(x, y int) float64 {
	if x < k.x0 || x >= k.x1 || y < k.y0 || y >= k.y1 {
		return 0
	}
	return k.gamma[(x-k.x0)+(y-k.y0)*k.stride]
}

func edgeOrientation(k, n *kernel, width, height int) (float64, float64) {
	ratio := func(x, y int) (float64, bool) {
		x = min(max(x, 0), width-1)
		y = min(max(y, 0), height-1)
		kg, ng := k.gammaAt(x, y), n.gammaAt(x, y)
		if kg+ng == 0 {
			return 0, false
		}
		return kg / (kg + ng), true
	}

	var ox, oy float64
	for y := k.y0; y < k.y1; y++ {
		for x := k.x0; x < k.x1; x++ {
			center, centerOK := ratio(x, y)
			left, leftOK := ratio(x-1, y)
			right, rightOK := ratio(x+1, y)
			switch {
			case leftOK && rightOK:
				ox += (right - left) / 2
			case centerOK && rightOK:
				ox += right - center
			case leftOK && centerOK:
				ox += center - left
			}
			top, topOK := ratio(x, y-1)
			bottom, bottomOK := ratio(x, y+1)
			switch {
			case topOK && bottomOK:
				oy += (bottom - top) / 2
			case centerOK && bottomOK:
				oy += bottom - center
			case topOK && centerOK:
				oy += center - top
			}
		}
	}
	return ox, oy
}

func converged(previous, kernels []kernel) bool {
	for i, k := range kernels {
		p := previous[i]
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
