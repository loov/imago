// Package perceptual implements SSIM-based perceptual image downscaling.
package perceptual

import (
	"errors"
	"math"

	"github.com/loov/imago/pix"
)

const varianceEpsilon = 1e-6

// Resize downsizes src using the closed-form SSIM optimization from
// Öztireli and Gross, "Perceptually Based Downscaling of Images" (2015).
// See https://doi.org/10.1145/2766891.
// Width and height must be positive and no larger than src in either dimension.
func Resize(src *pix.Image, width, height int) (*pix.Image, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("perceptual: empty image")
	}
	inputWidth, inputHeight := src.W, src.H
	if width <= 0 || height <= 0 || width > inputWidth || height > inputHeight {
		return nil, errors.New("perceptual: output dimensions must be positive and no larger than the input")
	}
	if width == inputWidth && height == inputHeight {
		return src.Clone(), nil
	}

	out := pix.New(width, height)
	scaleX := inputWidth / width
	if inputWidth%width != 0 {
		scaleX++
	}
	scaleY := inputHeight / height
	if inputHeight%height != 0 {
		scaleY++
	}
	preparedWidth, preparedHeight := width*scaleX, height*scaleY
	for i := range 4 {
		channel := src.Channel(i)
		if preparedWidth != inputWidth || preparedHeight != inputHeight {
			channel = resizeBicubic(channel, inputWidth, inputHeight, preparedWidth, preparedHeight)
		}
		out.SetChannel(i, downscale(channel, preparedWidth, width, height, scaleX, scaleY))
	}

	return out, nil
}

func downscale(h []float64, inputWidth, width, height, scaleX, scaleY int) []float64 {
	l := make([]float64, width*height)
	l2 := make([]float64, width*height)
	area := float64(scaleX * scaleY)
	for y := range height {
		for x := range width {
			var sum, sum2 float64
			for yy := range scaleY {
				row := (y*scaleY + yy) * inputWidth
				for xx := range scaleX {
					v := h[row+x*scaleX+xx]
					sum += v
					sum2 += v * v
				}
			}
			i := x + y*width
			l[i] = sum / area
			l2[i] = sum2 / area
		}
	}

	patchWidth, patchHeight := min(2, width), min(2, height)
	patchesWidth, patchesHeight := width-patchWidth+1, height-patchHeight+1
	d := make([]float64, width*height)
	counts := make([]float64, width*height)
	patchArea := float64(patchWidth * patchHeight)
	for y := range patchesHeight {
		for x := range patchesWidth {
			var mean, meanL2, meanH2 float64
			for yy := range patchHeight {
				for xx := range patchWidth {
					i := x + xx + (y+yy)*width
					mean += l[i]
					meanL2 += l[i] * l[i]
					meanH2 += l2[i]
				}
			}
			mean /= patchArea
			sl := meanL2/patchArea - mean*mean
			sh := meanH2/patchArea - mean*mean
			ratio := 0.0
			if sl >= varianceEpsilon {
				ratio = math.Sqrt(max(0, sh) / sl)
			}

			for yy := range patchHeight {
				for xx := range patchWidth {
					i := x + xx + (y+yy)*width
					d[i] += mean + ratio*(l[i]-mean)
					counts[i]++
				}
			}
		}
	}
	for i := range d {
		d[i] /= counts[i]
	}
	return d
}

func resizeBicubic(src []float64, srcWidth, srcHeight, width, height int) []float64 {
	dst := make([]float64, width*height)
	for y := range height {
		sy := (float64(y)+0.5)*float64(srcHeight)/float64(height) - 0.5
		y0 := int(math.Floor(sy))
		for x := range width {
			sx := (float64(x)+0.5)*float64(srcWidth)/float64(width) - 0.5
			x0 := int(math.Floor(sx))
			var value, weight float64
			for yy := y0 - 1; yy <= y0+2; yy++ {
				wy := cubic(sy - float64(yy))
				cy := min(max(yy, 0), srcHeight-1)
				for xx := x0 - 1; xx <= x0+2; xx++ {
					w := wy * cubic(sx-float64(xx))
					cx := min(max(xx, 0), srcWidth-1)
					value += w * src[cx+cy*srcWidth]
					weight += w
				}
			}
			dst[x+y*width] = value / weight
		}
	}
	return dst
}

func cubic(x float64) float64 {
	x = math.Abs(x)
	if x <= 1 {
		return (1.5*x-2.5)*x*x + 1
	}
	if x < 2 {
		return ((-0.5*x+2.5)*x-4)*x + 2
	}
	return 0
}
