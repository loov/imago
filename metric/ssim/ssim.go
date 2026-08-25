package ssim

import (
	"errors"
	"math"

	"github.com/loov/imago/pix"
)

const (
	windowSize = 11
	sigma      = 1.5
	k1, k2     = 0.01, 0.03
	c1         = (k1 * 1) * (k1 * 1)
	c2         = (k2 * 1) * (k2 * 1)
)

var msWeights = [5]float64{0.0448, 0.2856, 0.3001, 0.2363, 0.1333}

// SSIM returns the mean structural similarity index of a and b following
// Wang et al. 2004, https://doi.org/10.1109/TIP.2003.819861.
// It uses an 11x11 Gaussian window (σ=1.5) and requires both images to have
// the same size, at least 11x11.
func SSIM(a, b *pix.Image) (float64, error) {
	la, lb, w, h, err := lumas(a, b)
	if err != nil {
		return 0, err
	}
	if w < windowSize || h < windowSize {
		return 0, errors.New("ssim: image smaller than 11x11")
	}
	_, _, s := terms(la, lb, w, h, newScratch(w, h))
	return s, nil
}

// MSSSIM returns the multi-scale structural similarity index of a and b
// following Wang, Simoncelli, Bovik 2003, https://doi.org/10.1109/ACSSC.2003.1292216.
// It uses 5 scales, so both images must have the same size, at least 176x176.
// Negative contrast-structure terms are clamped to zero, so the result is in [0, 1].
func MSSSIM(a, b *pix.Image) (float64, error) {
	la, lb, w, h, err := lumas(a, b)
	if err != nil {
		return 0, err
	}
	if w>>(len(msWeights)-1) < windowSize || h>>(len(msWeights)-1) < windowSize {
		return 0, errors.New("ssim: image smaller than 176x176")
	}
	scratch := newScratch(w, h)
	result := 1.0
	for i, weight := range msWeights {
		l, cs, _ := terms(la, lb, w, h, scratch)
		if i == len(msWeights)-1 {
			result *= math.Pow(max(l, 0), weight)
		} else {
			halve(la, w, h)
			halve(lb, w, h)
			w, h = w/2, h/2
			la, lb = la[:w*h], lb[:w*h]
		}
		result *= math.Pow(max(cs, 0), weight)
	}
	if math.IsNaN(result) {
		return 0, errors.New("ssim: non-finite result")
	}
	return result, nil
}

func lumas(a, b *pix.Image) (la, lb []float64, w, h int, err error) {
	if a == nil || b == nil || a.W == 0 || a.H == 0 || b.W == 0 || b.H == 0 {
		return nil, nil, 0, 0, errors.New("ssim: empty image")
	}
	if a.W != b.W || a.H != b.H {
		return nil, nil, 0, 0, errors.New("ssim: image sizes differ")
	}
	return luma(a), luma(b), a.W, a.H, nil
}

func luma(src *pix.Image) []float64 {
	dst := make([]float64, src.W*src.H)
	for i := range dst {
		r, g, b, a := src.Pix[4*i], src.Pix[4*i+1], src.Pix[4*i+2], src.Pix[4*i+3]
		if a > 0 {
			r, g, b = r/a, g/a, b/a
		}
		dst[i] = 0.299*r + 0.587*g + 0.114*b
	}
	return dst
}

// newScratch allocates the horizontal-pass buffer for the five moments,
// sized for the largest scale and reused by smaller ones.
func newScratch(w, h int) []float64 {
	return make([]float64, 5*(w-windowSize+1)*h)
}

// terms returns the means over the valid SSIM map of the luminance term,
// the contrast-structure term, and their product (the SSIM index).
//
// The five moments μa, μb, E[a²], E[b²], E[ab] are blurred with a separable
// 11x11 Gaussian: the horizontal pass computes the products on the fly into
// scratch (five interleaved moments per valid pixel, 5*vw*h), and the vertical pass accumulates the
// means directly, so no moment or result planes are ever materialized.
func terms(a, b []float64, w, h int, scratch []float64) (l, cs, s float64) {
	kernel := gaussian()
	vw, vh := w-windowSize+1, h-windowSize+1
	tmp := scratch[:5*vw*h] // interleaved: tmp[5*i+m] is moment m at valid pixel i
	for y := range h {
		for x := range vw {
			var v [5]float64
			for k, kv := range kernel {
				i := x + k + y*w
				ai, bi := a[i], b[i]
				v[0] += kv * ai
				v[1] += kv * bi
				v[2] += kv * ai * ai
				v[3] += kv * bi * bi
				v[4] += kv * ai * bi
			}
			copy(tmp[5*(x+y*vw):], v[:])
		}
	}
	for y := range vh {
		for x := range vw {
			var v [5]float64
			for k, kv := range kernel {
				t := tmp[5*(x+(y+k)*vw):][:5]
				v[0] += kv * t[0]
				v[1] += kv * t[1]
				v[2] += kv * t[2]
				v[3] += kv * t[3]
				v[4] += kv * t[4]
			}
			ma, mb, maa, mbb, mab := v[0], v[1], v[2], v[3], v[4]
			covAB := mab - ma*mb
			varA := maa - ma*ma
			varB := mbb - mb*mb
			li := (2*ma*mb + c1) / (ma*ma + mb*mb + c1)
			csi := (2*covAB + c2) / (varA + varB + c2)
			l, cs, s = l+li, cs+csi, s+li*csi
		}
	}
	fn := float64(vw * vh)
	return l / fn, cs / fn, s / fn
}

func gaussian() (kernel [windowSize]float64) {
	sum := 0.0
	for i := range kernel {
		d := float64(i - windowSize/2)
		kernel[i] = math.Exp(-d * d / (2 * sigma * sigma))
		sum += kernel[i]
	}
	for i := range kernel {
		kernel[i] /= sum
	}
	return kernel
}

// halve downsamples in place by 2 with a 2x2 average, dropping any odd
// trailing row or column. The result occupies the front (w/2)*(h/2) of src.
func halve(src []float64, w, h int) {
	hw, hh := w/2, h/2
	for y := range hh {
		for x := range hw {
			i := 2*x + 2*y*w
			src[x+y*hw] = (src[i] + src[i+1] + src[i+w] + src[i+w+1]) / 4
		}
	}
}
