// Package sketch cleans up photographed/scanned pencil sketches by flattening
// uneven paper lighting to a uniform white, a port of
// https://github.com/loov/sketchure (cleanup.ByBase).
package sketch

import (
	"errors"
	"image"
	"image/color"

	"github.com/loov/imago/filter"
)

// Options tunes Clean. The zero value uses the sketchure defaults.
type Options struct {
	Whiteness float64 // lightest output value, 0..100; >100 clips highlights. 0 → 100.
	LineWidth int     // max line width in px; 0 → width/20 (min 1)
	KeepColor bool    // keep chroma instead of desaturating (default false = desaturate, matching sketchure)
}

// Clean normalizes the paper of a sketch to uniform white while keeping lines dark.
//
// Pipeline, working on Rec.601 YCbCr luma L:
//  1. 3×3 median to remove hot pixels;
//  2. base = erode(LineWidth) then blur(LineWidth) of L, an estimate of the paper;
//  3. L' = white + (L − base)·white/mean(base), clamped to 0..255,
//     where white = Whiteness·255/100;
//  4. unless KeepColor, Cb = Cr = 128 (desaturate).
//
// The result is a new *image.NRGBA with bounds (0,0)-(w,h); alpha is preserved.
func Clean(src *image.NRGBA, opts Options) (*image.NRGBA, error) {
	if src == nil {
		return nil, errors.New("sketch: nil image")
	}
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("sketch: empty image")
	}
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			m.SetNRGBA(x, y, src.NRGBAAt(src.Rect.Min.X+x, src.Rect.Min.Y+y))
		}
	}
	if opts.Whiteness == 0 {
		opts.Whiteness = 100
	}
	if opts.LineWidth <= 0 {
		opts.LineWidth = max(w/20, 1)
	}

	L := filter.NewChannel(w, h)
	cb := make([]byte, w*h)
	cr := make([]byte, w*h)
	for p := range w * h {
		L.Data[p], cb[p], cr[p] = color.RGBToYCbCr(m.Pix[4*p], m.Pix[4*p+1], m.Pix[4*p+2])
	}

	L.Median(1)
	base := L.Clone()
	base.Erode(opts.LineWidth)
	base.Blur(opts.LineWidth)

	white := opts.Whiteness * 255.0 / 100.0
	invspan := 1.0 / (base.Average() / white)
	for i, lv := range L.Data {
		r := int(white + (float64(lv)-float64(base.Data[i]))*invspan)
		L.Data[i] = byte(min(max(r, 0), 0xFF))
	}

	for p := range w * h {
		c1, c2 := cb[p], cr[p]
		if !opts.KeepColor {
			c1, c2 = 128, 128
		}
		m.Pix[4*p], m.Pix[4*p+1], m.Pix[4*p+2] = color.YCbCrToRGB(L.Data[p], c1, c2)
	}
	return m, nil
}
