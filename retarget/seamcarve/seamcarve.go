// Package seamcarve implements content-aware image shrinking by seam removal.
package seamcarve

import (
	"errors"
	"image"
	"image/color"
)

// Resize shrinks src by repeatedly removing minimum-energy seams, following
// Avidan and Shamir, "Seam Carving for Content-Aware Image Resizing"
// (SIGGRAPH 2007). See https://doi.org/10.1145/1275808.1276390.
// Width and height must be positive and no larger than src in either dimension.
func Resize(src *image.NRGBA, width, height int) (*image.NRGBA, error) {
	if src == nil {
		return nil, errors.New("seamcarve: nil image")
	}

	inputWidth, inputHeight := src.Rect.Dx(), src.Rect.Dy()
	if inputWidth <= 0 || inputHeight <= 0 {
		return nil, errors.New("seamcarve: empty image")
	}
	if width <= 0 || height <= 0 || width > inputWidth || height > inputHeight {
		return nil, errors.New("seamcarve: output dimensions must be positive and no larger than the input")
	}

	pix := make([]color.NRGBA, 0, inputWidth*inputHeight)
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			pix = append(pix, src.NRGBAAt(x, y))
		}
	}

	for w := inputWidth; w > width; w-- {
		pix = removeSeam(pix, w, inputHeight)
	}
	pix = transpose(pix, width, inputHeight)
	for h := inputHeight; h > height; h-- {
		pix = removeSeam(pix, h, width)
	}
	pix = transpose(pix, height, width)

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i, c := range pix {
		dst.Pix[4*i+0], dst.Pix[4*i+1], dst.Pix[4*i+2], dst.Pix[4*i+3] = c.R, c.G, c.B, c.A
	}
	return dst, nil
}

// removeSeam removes one minimum-energy vertical seam from a w×h image.
// ponytail: energy is recomputed from scratch each call (O(w·h) per seam);
// update only the columns adjacent to the removed seam if this gets slow.
func removeSeam(pix []color.NRGBA, w, h int) []color.NRGBA {
	at := func(x, y int) color.NRGBA {
		return pix[min(max(x, 0), w-1)+min(max(y, 0), h-1)*w]
	}
	// Alpha counts as detail: compare premultiplied color plus alpha, so fully
	// transparent pixels are equal regardless of their hidden RGB.
	diff := func(a, b color.NRGBA) int {
		pa, pb := premul(a), premul(b)
		return abs(pa[0]-pb[0]) + abs(pa[1]-pb[1]) + abs(pa[2]-pb[2]) + abs(pa[3]-pb[3])
	}

	// Cumulative minimum energy (e1) and the parent column for backtracking.
	cost := make([]int, w*h)
	from := make([]int, w*h)
	for y := range h {
		for x := range w {
			e := diff(at(x-1, y), at(x+1, y)) + diff(at(x, y-1), at(x, y+1))
			best := 0
			if y > 0 {
				best = cost[x+(y-1)*w]
				from[x+y*w] = x
				for _, nx := range [2]int{x - 1, x + 1} {
					if nx >= 0 && nx < w && cost[nx+(y-1)*w] < best {
						best = cost[nx+(y-1)*w]
						from[x+y*w] = nx
					}
				}
			}
			cost[x+y*w] = best + e
		}
	}

	x := 0
	for nx := range w {
		if cost[nx+(h-1)*w] < cost[x+(h-1)*w] {
			x = nx
		}
	}
	out := make([]color.NRGBA, 0, (w-1)*h)
	for y := h - 1; y >= 0; y-- {
		row := pix[y*w : (y+1)*w]
		// Build rows in reverse then fix order below.
		out = append(out, row[:x]...)
		out = append(out, row[x+1:]...)
		x = from[x+y*w]
	}
	for y := range h / 2 {
		a, b := out[y*(w-1):(y+1)*(w-1)], out[(h-1-y)*(w-1):(h-y)*(w-1)]
		for i := range a {
			a[i], b[i] = b[i], a[i]
		}
	}
	return out
}

func transpose(pix []color.NRGBA, w, h int) []color.NRGBA {
	out := make([]color.NRGBA, len(pix))
	for y := range h {
		for x := range w {
			out[y+x*h] = pix[x+y*w]
		}
	}
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func premul(c color.NRGBA) [4]int {
	a := int(c.A)
	return [4]int{int(c.R) * a / 255, int(c.G) * a / 255, int(c.B) * a / 255, a}
}
