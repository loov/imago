package box

import (
	"errors"

	"github.com/loov/imago/pix"
)

// Resize downsizes src so that every output pixel is the average of its exact
// source rectangle, weighting partially covered source pixels by their
// fractional coverage. Width and height must be positive and no larger than
// src in either dimension.
func Resize(src *pix.Image, width, height int) (*pix.Image, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("box: empty image")
	}
	inputWidth, inputHeight := src.W, src.H
	if width <= 0 || height <= 0 || width > inputWidth || height > inputHeight {
		return nil, errors.New("box: output dimensions must be positive and no larger than the input")
	}
	if width == inputWidth && height == inputHeight {
		return src.Clone(), nil
	}

	scaleX := float64(inputWidth) / float64(width)
	scaleY := float64(inputHeight) / float64(height)
	out := pix.New(width, height)
	for y := range height {
		y0, y1 := float64(y)*scaleY, float64(y+1)*scaleY
		for x := range width {
			x0, x1 := float64(x)*scaleX, float64(x+1)*scaleX
			var sum [4]float64
			for sy := int(y0); sy < inputHeight && float64(sy) < y1; sy++ {
				cy := min(y1, float64(sy+1)) - max(y0, float64(sy))
				for sx := int(x0); sx < inputWidth && float64(sx) < x1; sx++ {
					w := cy * (min(x1, float64(sx+1)) - max(x0, float64(sx)))
					i := 4 * (sx + sy*inputWidth)
					for c := range 4 {
						sum[c] += w * src.Pix[i+c]
					}
				}
			}
			area := scaleX * scaleY
			out.Set(x, y, sum[0]/area, sum[1]/area, sum[2]/area, sum[3]/area)
		}
	}
	return out, nil
}
