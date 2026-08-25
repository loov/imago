package pixelart

import (
	"image"
	"image/color"
	"math"
)

// pixel is a source color with its premultiplied form k (R,G,B,A), which all
// comparisons and distances use: alpha counts as detail, and fully transparent
// pixels compare equal regardless of their hidden RGB.
type pixel struct {
	c color.NRGBA
	k [4]int
}

func premul(c color.NRGBA) pixel {
	a := int(c.A)
	return pixel{c, [4]int{int(c.R) * a / 255, int(c.G) * a / 255, int(c.B) * a / 255, a}}
}

// grid reads src with edge-clamped access.
type grid struct {
	pix  []pixel
	w, h int
}

// read returns nil for a nil or empty src.
func read(src *image.NRGBA) *grid {
	if src == nil || src.Rect.Empty() {
		return nil
	}
	g := &grid{w: src.Rect.Dx(), h: src.Rect.Dy()}
	g.pix = make([]pixel, g.w*g.h)
	for y := range g.h {
		for x := range g.w {
			g.pix[x+y*g.w] = premul(src.NRGBAAt(src.Rect.Min.X+x, src.Rect.Min.Y+y))
		}
	}
	return g
}

func (g *grid) at(x, y int) pixel {
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
			if c.k == a.k && c.k != d.k && a.k != b.k {
				e0 = a
			}
			if a.k == b.k && a.k != c.k && b.k != d.k {
				e1 = b
			}
			if d.k == c.k && d.k != b.k && c.k != a.k {
				e2 = c
			}
			if b.k == d.k && b.k != a.k && d.k != c.k {
				e3 = d
			}
			dst.SetNRGBA(2*x, 2*y, e0.c)
			dst.SetNRGBA(2*x+1, 2*y, e1.c)
			dst.SetNRGBA(2*x, 2*y+1, e2.c)
			dst.SetNRGBA(2*x+1, 2*y+1, e3.c)
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
			out := [9]pixel{e, e, e, e, e, e, e, e, e}
			if d.k == b.k && b.k != f.k && d.k != h.k {
				out[0] = d
			}
			if (d.k == b.k && b.k != f.k && d.k != h.k && e.k != c.k) || (b.k == f.k && b.k != d.k && f.k != h.k && e.k != a.k) {
				out[1] = b
			}
			if b.k == f.k && b.k != d.k && f.k != h.k {
				out[2] = f
			}
			if (d.k == b.k && b.k != f.k && d.k != h.k && e.k != gg.k) || (d.k == h.k && d.k != b.k && h.k != f.k && e.k != a.k) {
				out[3] = d
			}
			if (b.k == f.k && b.k != d.k && f.k != h.k && e.k != i.k) || (h.k == f.k && d.k != h.k && b.k != f.k && e.k != c.k) {
				out[5] = f
			}
			if d.k == h.k && d.k != b.k && h.k != f.k {
				out[6] = d
			}
			if (d.k == h.k && d.k != b.k && h.k != f.k && e.k != i.k) || (h.k == f.k && d.k != h.k && b.k != f.k && e.k != gg.k) {
				out[7] = h
			}
			if h.k == f.k && d.k != h.k && b.k != f.k {
				out[8] = f
			}
			for k, v := range out {
				dst.SetNRGBA(3*x+k%3, 3*y+k/3, v.c)
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
			var n [5][5]pixel
			for dy := range 5 {
				for dx := range 5 {
					n[dy][dx] = g.at(x+dx-2, y+dy-2)
				}
			}
			for k := range 4 {
				if k > 0 {
					var r [5][5]pixel
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
				if e.k != f.k && e.k != h.k && edge < interior {
					nc := h
					if dist(e, f) <= dist(e, h) {
						nc = f
					}
					out = blend(e, nc)
				}
				dst.SetNRGBA(2*x+corner[k][0], 2*y+corner[k][1], out.c)
			}
		}
	}
	return dst
}

// dist is the YUV-weighted distance from the xBR reference (Y weight 48,
// U 7, V 6) over premultiplied color, plus alpha difference with the Y weight.
func dist(a, b pixel) float64 {
	ya, ua, va := yuv(a.k)
	yb, ub, vb := yuv(b.k)
	return 48*math.Abs(ya-yb) + 7*math.Abs(ua-ub) + 6*math.Abs(va-vb) + 48*math.Abs(float64(a.k[3]-b.k[3]))
}

func yuv(k [4]int) (y, u, v float64) {
	r, g, b := float64(k[0]), float64(k[1]), float64(k[2])
	return 0.299*r + 0.587*g + 0.114*b,
		-0.169*r - 0.331*g + 0.5*b,
		0.5*r - 0.419*g - 0.081*b
}

// blend averages in premultiplied space and un-premultiplies the result.
func blend(a, b pixel) pixel {
	var k [4]int
	for i := range k {
		k[i] = (a.k[i] + b.k[i] + 1) / 2
	}
	var c color.NRGBA
	if k[3] > 0 {
		un := func(v int) uint8 { return uint8(min((v*255+k[3]/2)/k[3], 255)) }
		c = color.NRGBA{R: un(k[0]), G: un(k[1]), B: un(k[2]), A: uint8(k[3])}
	}
	return pixel{c, k}
}
