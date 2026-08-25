// Package ssim implements the SSIM and MS-SSIM image similarity indices.
//
// Both indices compare images on luma only: Rec. 601 Y computed from the
// sRGB-encoded straight (un-premultiplied) RGB channels in [0, 1]. Alpha is
// otherwise ignored.
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
	_, _, s := terms(la, lb, w, h)
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
	result := 1.0
	for i, weight := range msWeights {
		l, cs, _ := terms(la, lb, w, h)
		if i == len(msWeights)-1 {
			result *= math.Pow(max(l, 0), weight)
		} else {
			la, lb, w, h = halve(la, w, h), halve(lb, w, h), w/2, h/2
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

// terms returns the means over the valid SSIM map of the luminance term,
// the contrast-structure term, and their product (the SSIM index).
func terms(a, b []float64, w, h int) (l, cs, s float64) {
	ab := make([]float64, len(a))
	aa := make([]float64, len(a))
	bb := make([]float64, len(a))
	for i := range a {
		ab[i], aa[i], bb[i] = a[i]*b[i], a[i]*a[i], b[i]*b[i]
	}
	ma, mb := blur(a, w, h), blur(b, w, h)
	mab, maa, mbb := blur(ab, w, h), blur(aa, w, h), blur(bb, w, h)
	for i := range ma {
		covAB := mab[i] - ma[i]*mb[i]
		varA := maa[i] - ma[i]*ma[i]
		varB := mbb[i] - mb[i]*mb[i]
		li := (2*ma[i]*mb[i] + c1) / (ma[i]*ma[i] + mb[i]*mb[i] + c1)
		csi := (2*covAB + c2) / (varA + varB + c2)
		l, cs, s = l+li, cs+csi, s+li*csi
	}
	n := float64(len(ma))
	return l / n, cs / n, s / n
}

// blur applies the separable 11x11 Gaussian window and returns the
// (w-10)x(h-10) valid region.
func blur(src []float64, w, h int) []float64 {
	var kernel [windowSize]float64
	sum := 0.0
	for i := range kernel {
		d := float64(i - windowSize/2)
		kernel[i] = math.Exp(-d * d / (2 * sigma * sigma))
		sum += kernel[i]
	}
	for i := range kernel {
		kernel[i] /= sum
	}

	vw, vh := w-windowSize+1, h-windowSize+1
	tmp := make([]float64, vw*h) // horizontal pass
	for y := range h {
		for x := range vw {
			v := 0.0
			for k, kv := range kernel {
				v += kv * src[x+k+y*w]
			}
			tmp[x+y*vw] = v
		}
	}
	dst := make([]float64, vw*vh) // vertical pass
	for y := range vh {
		for x := range vw {
			v := 0.0
			for k, kv := range kernel {
				v += kv * tmp[x+(y+k)*vw]
			}
			dst[x+y*vw] = v
		}
	}
	return dst
}

// halve downsamples by 2 with a 2x2 average, dropping any odd trailing row or column.
func halve(src []float64, w, h int) []float64 {
	hw, hh := w/2, h/2
	dst := make([]float64, hw*hh)
	for y := range hh {
		for x := range hw {
			i := 2*x + 2*y*w
			dst[x+y*hw] = (src[i] + src[i+1] + src[i+w] + src[i+w+1]) / 4
		}
	}
	return dst
}
