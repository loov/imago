// Package pixelart implements pixel-art upscalers: Scale2x, Scale3x and xBR.
package pixelart

import (
	"image"
	"image/color"
	"math"
)

// grid reads src with edge-clamped access.
type grid struct {
	pix  []color.NRGBA
	w, h int
}

// read returns nil for a nil or empty src.
func read(src *image.NRGBA) *grid {
	if src == nil || src.Rect.Empty() {
		return nil
	}
	g := &grid{w: src.Rect.Dx(), h: src.Rect.Dy()}
	g.pix = make([]color.NRGBA, g.w*g.h)
	for y := range g.h {
		for x := range g.w {
			g.pix[x+y*g.w] = src.NRGBAAt(src.Rect.Min.X+x, src.Rect.Min.Y+y)
		}
	}
	return g
}

func (g *grid) at(x, y int) color.NRGBA {
	return g.pix[min(max(x, 0), g.w-1)+min(max(y, 0), g.h-1)*g.w]
}

// Scale2x implements AdvMAME2x / EPX by Andrea Mazzoleni.
// See https://www.scale2x.it/algorithm.
func Scale2x(src *image.NRGBA) *image.NRGBA {
	g := read(src)
	if g == nil {
		return image.NewNRGBA(image.Rect(0, 0, 0, 0))
	}
	dst := image.NewNRGBA(image.Rect(0, 0, 2*g.w, 2*g.h))
	for y := range g.h {
		for x := range g.w {
			a, b, c, d, p := g.at(x, y-1), g.at(x+1, y), g.at(x-1, y), g.at(x, y+1), g.at(x, y)
			e0, e1, e2, e3 := p, p, p, p
			if c == a && c != d && a != b {
				e0 = a
			}
			if a == b && a != c && b != d {
				e1 = b
			}
			if d == c && d != b && c != a {
				e2 = c
			}
			if b == d && b != a && d != c {
				e3 = d
			}
			dst.SetNRGBA(2*x, 2*y, e0)
			dst.SetNRGBA(2*x+1, 2*y, e1)
			dst.SetNRGBA(2*x, 2*y+1, e2)
			dst.SetNRGBA(2*x+1, 2*y+1, e3)
		}
	}
	return dst
}

// Scale3x implements AdvMAME3x by Andrea Mazzoleni.
// See https://www.scale2x.it/algorithm.
func Scale3x(src *image.NRGBA) *image.NRGBA {
	g := read(src)
	if g == nil {
		return image.NewNRGBA(image.Rect(0, 0, 0, 0))
	}
	dst := image.NewNRGBA(image.Rect(0, 0, 3*g.w, 3*g.h))
	for y := range g.h {
		for x := range g.w {
			a, b, c := g.at(x-1, y-1), g.at(x, y-1), g.at(x+1, y-1)
			d, e, f := g.at(x-1, y), g.at(x, y), g.at(x+1, y)
			gg, h, i := g.at(x-1, y+1), g.at(x, y+1), g.at(x+1, y+1)
			out := [9]color.NRGBA{e, e, e, e, e, e, e, e, e}
			if d == b && b != f && d != h {
				out[0] = d
			}
			if (d == b && b != f && d != h && e != c) || (b == f && b != d && f != h && e != a) {
				out[1] = b
			}
			if b == f && b != d && f != h {
				out[2] = f
			}
			if (d == b && b != f && d != h && e != gg) || (d == h && d != b && h != f && e != a) {
				out[3] = d
			}
			if (b == f && b != d && f != h && e != i) || (h == f && d != h && b != f && e != c) {
				out[5] = f
			}
			if d == h && d != b && h != f {
				out[6] = d
			}
			if (d == h && d != b && h != f && e != i) || (h == f && d != h && b != f && e != gg) {
				out[7] = h
			}
			if h == f && d != h && b != f {
				out[8] = f
			}
			for k, v := range out {
				dst.SetNRGBA(3*x+k%3, 3*y+k/3, v)
			}
		}
	}
	return dst
}

// XBR2x implements Hyllian's xBR 2x kernel, level 1 (edge detection and
// interpolation along the detected edge, without the level 2/3 refinements).
// See https://forums.libretro.com/t/xbr-algorithm-tutorial/123.
//
// For each of the four output pixels the 5x5 neighborhood is rotated so the
// pixel is at the bottom-right corner:
//
//	   A1 B1 C1
//	A0 A  B  C  C4
//	D0 D  E  F  F4
//	G0 G  H  I  I4
//	   G5 H5 I5
//
// The edge runs between the ends of the H-F diagonal when the "edge" weight
// e = d(E,C)+d(E,G)+d(I,H5)+d(I,F4)+4*d(H,F) is smaller than the "interior"
// weight i = d(H,D)+d(H,I5)+d(F,I4)+d(F,B)+4*d(E,I). The output pixel then
// becomes a 50% blend of E with whichever of F or H is closer to E.
func XBR2x(src *image.NRGBA) *image.NRGBA {
	g := read(src)
	if g == nil {
		return image.NewNRGBA(image.Rect(0, 0, 0, 0))
	}
	dst := image.NewNRGBA(image.Rect(0, 0, 2*g.w, 2*g.h))
	// Output offset of the bottom-right corner after k clockwise rotations.
	corner := [4][2]int{{1, 1}, {1, 0}, {0, 0}, {0, 1}}
	for y := range g.h {
		for x := range g.w {
			var n [5][5]color.NRGBA
			for dy := range 5 {
				for dx := range 5 {
					n[dy][dx] = g.at(x+dx-2, y+dy-2)
				}
			}
			for k := range 4 {
				if k > 0 {
					var r [5][5]color.NRGBA
					for dy := range 5 {
						for dx := range 5 {
							r[dy][dx] = n[4-dx][dy]
						}
					}
					n = r
				}
				e, out := n[2][2], n[2][2]
				b, c, d, f, gg, h, i := n[1][2], n[1][3], n[2][1], n[2][3], n[3][1], n[3][2], n[3][3]
				f4, i4, i5, h5 := n[2][4], n[3][4], n[4][3], n[4][2]
				edge := dist(e, c) + dist(e, gg) + dist(i, h5) + dist(i, f4) + 4*dist(h, f)
				interior := dist(h, d) + dist(h, i5) + dist(f, i4) + dist(f, b) + 4*dist(e, i)
				if e != f && e != h && edge < interior {
					nc := h
					if dist(e, f) <= dist(e, h) {
						nc = f
					}
					out = blend(e, nc)
				}
				dst.SetNRGBA(2*x+corner[k][0], 2*y+corner[k][1], out)
			}
		}
	}
	return dst
}

// dist is the YUV-weighted color distance from the xBR reference
// (Y weight 48, U 7, V 6). Alpha difference is counted with the Y weight.
func dist(a, b color.NRGBA) float64 {
	ya, ua, va := yuv(a)
	yb, ub, vb := yuv(b)
	return 48*math.Abs(ya-yb) + 7*math.Abs(ua-ub) + 6*math.Abs(va-vb) + 48*math.Abs(float64(a.A)-float64(b.A))
}

func yuv(c color.NRGBA) (y, u, v float64) {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	return 0.299*r + 0.587*g + 0.114*b,
		-0.169*r - 0.331*g + 0.5*b,
		0.5*r - 0.419*g - 0.081*b
}

func blend(a, b color.NRGBA) color.NRGBA {
	return color.NRGBA{
		R: uint8((uint16(a.R) + uint16(b.R) + 1) / 2),
		G: uint8((uint16(a.G) + uint16(b.G) + 1) / 2),
		B: uint8((uint16(a.B) + uint16(b.B) + 1) / 2),
		A: uint8((uint16(a.A) + uint16(b.A) + 1) / 2),
	}
}
