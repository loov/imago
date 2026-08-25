package dither

import (
	"image"
	"image/color"
	"math"
	"math/bits"
)

// Ordered maps src onto p with a Bayer matrix of the given power-of-two size
// (2, 4, 8, ...; other values are rounded up), returning a paletted image at
// origin (0,0). strength in 0..1 scales the threshold offset applied before
// nearest-color lookup; 0 disables dithering. Transparent pixels (alpha 0)
// take the palette's first fully transparent entry, or index 0 if none.
func Ordered(src *image.NRGBA, p color.Palette, size int, strength float32) *image.Paletted {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	dst := image.NewPaletted(image.Rect(0, 0, w, h), p)
	if len(p) == 0 {
		return dst
	}
	m := Bayer(size)
	n := len(m)
	// Spread is the color step the dither covers: with k palette levels per
	// channel a full pattern spans roughly 255/(k^(1/3)). Use the palette
	// size as a rough level count.
	levels := max(1, int(math.Cbrt(float64(len(p)))))
	spread := strength * 255 / float32(levels)
	transparent := transparentIndex(p)
	for y := range h {
		for x := range w {
			c := src.NRGBAAt(src.Rect.Min.X+x, src.Rect.Min.Y+y)
			if c.A == 0 {
				dst.Pix[y*dst.Stride+x] = uint8(transparent)
				continue
			}
			t := int(math.Round(float64((m[y%n][x%n] - 0.5) * spread)))
			shifted := color.NRGBA{clamp8(int(c.R) + t), clamp8(int(c.G) + t), clamp8(int(c.B) + t), c.A}
			dst.Pix[y*dst.Stride+x] = uint8(p.Index(shifted))
		}
	}
	return dst
}

// Bayer returns the normalized n×n Bayer threshold matrix with entries in
// (0, 1), for n the smallest power of two >= size (at least 2).
func Bayer(size int) [][]float32 {
	n := 2
	if size > 2 {
		n = 1 << bits.Len(uint(size-1))
	}
	m := [][]int{{0, 2}, {3, 1}}
	for len(m) < n {
		k := len(m)
		next := make([][]int, 2*k)
		for y := range next {
			next[y] = make([]int, 2*k)
		}
		for y := range k {
			for x := range k {
				v := 4 * m[y][x]
				next[y][x] = v
				next[y][x+k] = v + 2
				next[y+k][x] = v + 3
				next[y+k][x+k] = v + 1
			}
		}
		m = next
	}
	out := make([][]float32, n)
	for y := range out {
		out[y] = make([]float32, n)
		for x := range out[y] {
			out[y][x] = (float32(m[y][x]) + 0.5) / float32(n*n)
		}
	}
	return out
}

func transparentIndex(p color.Palette) int {
	for i, c := range p {
		if _, _, _, a := c.RGBA(); a == 0 {
			return i
		}
	}
	return 0
}

func clamp8(v int) uint8 { return uint8(min(max(v, 0), 255)) }
