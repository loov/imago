package quant

import (
	"image"
	"image/color"
)

// Wu returns up to n colors by Xiaolin Wu's variance-minimizing quantizer
// (Graphics Gems II, 1991): colors are binned into a 32^3 histogram with
// cumulative moments, and the box with the largest weighted variance is cut
// along the axis and position that minimize the resulting variance. n < 1
// returns nil; an image whose colors fill fewer than n boxes returns fewer.
func Wu(src *image.NRGBA, n int) color.Palette {
	if n < 1 {
		return nil
	}
	const bits, side = 5, 33 // 32 bins per channel plus a zero row for prefix sums
	var wt, mr, mg, mb, m2 [side][side][side]float64
	hist := histogram(src)
	if len(hist) == 0 {
		return nil
	}
	for k, c := range hist {
		r, g, b := unpack(k)
		i, j, l := int(r>>(8-bits))+1, int(g>>(8-bits))+1, int(b>>(8-bits))+1
		w := float64(c)
		wt[i][j][l] += w
		mr[i][j][l] += w * float64(r)
		mg[i][j][l] += w * float64(g)
		mb[i][j][l] += w * float64(b)
		m2[i][j][l] += w * (float64(r)*float64(r) + float64(g)*float64(g) + float64(b)*float64(b))
	}
	// Cumulative moments so that any box sum is an inclusion-exclusion of 8 corners.
	for _, m := range []*[side][side][side]float64{&wt, &mr, &mg, &mb, &m2} {
		for i := 1; i < side; i++ {
			for j := 1; j < side; j++ {
				for l := 1; l < side; l++ {
					m[i][j][l] += m[i-1][j][l] + m[i][j-1][l] + m[i][j][l-1] -
						m[i-1][j-1][l] - m[i-1][j][l-1] - m[i][j-1][l-1] + m[i-1][j-1][l-1]
				}
			}
		}
	}

	type box struct{ r0, r1, g0, g1, b0, b1 int } // half-open on the upper side, in bin units
	vol := func(m *[side][side][side]float64, c box) float64 {
		return m[c.r1][c.g1][c.b1] - m[c.r1][c.g1][c.b0] - m[c.r1][c.g0][c.b1] + m[c.r1][c.g0][c.b0] -
			m[c.r0][c.g1][c.b1] + m[c.r0][c.g1][c.b0] + m[c.r0][c.g0][c.b1] - m[c.r0][c.g0][c.b0]
	}
	// variance of the box's colors, weighted by count.
	variance := func(c box) float64 {
		w := vol(&wt, c)
		if w == 0 {
			return 0
		}
		r, g, b := vol(&mr, c), vol(&mg, c), vol(&mb, c)
		return vol(&m2, c) - (r*r+g*g+b*b)/w
	}
	// bottom and top are the moments of the box's faces at the low and at a
	// candidate cut position along axis.
	moments := func(m *[side][side][side]float64, c box, axis, pos int) float64 {
		switch axis {
		case 0:
			return m[pos][c.g1][c.b1] - m[pos][c.g1][c.b0] - m[pos][c.g0][c.b1] + m[pos][c.g0][c.b0]
		case 1:
			return m[c.r1][pos][c.b1] - m[c.r1][pos][c.b0] - m[c.r0][pos][c.b1] + m[c.r0][pos][c.b0]
		default:
			return m[c.r1][c.g1][pos] - m[c.r1][c.g0][pos] - m[c.r0][c.g1][pos] + m[c.r0][c.g0][pos]
		}
	}
	// maximize finds the cut along axis that maximizes the sum of the two
	// halves' (sum^2 / weight), which is what minimizes total variance.
	maximize := func(c box, axis, lo, hi int) (best float64, cut int) {
		wholeW, wholeR, wholeG, wholeB := vol(&wt, c), vol(&mr, c), vol(&mg, c), vol(&mb, c)
		baseW, baseR, baseG, baseB := moments(&wt, c, axis, lo), moments(&mr, c, axis, lo), moments(&mg, c, axis, lo), moments(&mb, c, axis, lo)
		best, cut = -1, -1
		for i := lo + 1; i < hi; i++ {
			w := moments(&wt, c, axis, i) - baseW
			r := moments(&mr, c, axis, i) - baseR
			g := moments(&mg, c, axis, i) - baseG
			b := moments(&mb, c, axis, i) - baseB
			if w == 0 {
				continue
			}
			t := (r*r + g*g + b*b) / w
			w2 := wholeW - w
			if w2 == 0 {
				continue
			}
			r2, g2, b2 := wholeR-r, wholeG-g, wholeB-b
			t += (r2*r2 + g2*g2 + b2*b2) / w2
			if t > best {
				best, cut = t, i
			}
		}
		return best, cut
	}
	cutBox := func(c box) (a, b box, ok bool) {
		var bestScore = -1.0
		bestAxis, bestCut := -1, -1
		for axis, lohi := range [3][2]int{{c.r0, c.r1}, {c.g0, c.g1}, {c.b0, c.b1}} {
			if s, cut := maximize(c, axis, lohi[0], lohi[1]); s > bestScore {
				bestScore, bestAxis, bestCut = s, axis, cut
			}
		}
		if bestCut < 0 {
			return c, c, false
		}
		a, b = c, c
		switch bestAxis {
		case 0:
			a.r1, b.r0 = bestCut, bestCut
		case 1:
			a.g1, b.g0 = bestCut, bestCut
		default:
			a.b1, b.b0 = bestCut, bestCut
		}
		return a, b, true
	}

	boxes := []box{{0, side - 1, 0, side - 1, 0, side - 1}}
	vars := []float64{variance(boxes[0])}
	for len(boxes) < n {
		next, nextVar := -1, 0.0
		for i, v := range vars {
			if v > nextVar {
				next, nextVar = i, v
			}
		}
		if next < 0 {
			break
		}
		a, b, ok := cutBox(boxes[next])
		if !ok {
			vars[next] = 0
			continue
		}
		boxes[next], vars[next] = a, variance(a)
		boxes = append(boxes, b)
		vars = append(vars, variance(b))
	}

	p := make(color.Palette, 0, len(boxes))
	for _, c := range boxes {
		w := vol(&wt, c)
		if w == 0 {
			continue
		}
		p = append(p, color.NRGBA{
			uint8(vol(&mr, c)/w + 0.5), uint8(vol(&mg, c)/w + 0.5), uint8(vol(&mb, c)/w + 0.5), 255,
		})
	}
	return p
}
