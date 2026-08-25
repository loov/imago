package pixelate

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/loov/imago/chroma"
	"github.com/loov/imago/pix"
)

// Parameters from the paper, section 4.
const (
	spatialWeight    = 45.0 // m: position weight in the SLIC distance
	initialTempScale = 1.0  // T0 = scale * sqrt(2 * largest eigenvalue)
	finalTemp        = 1.0  // Tf
	coolingRate      = 0.7  // alpha
	splitDistance    = 0.25 // εcluster: sub-clusters this far apart split
	perturbation     = 0.5  // εpalette: initial sub-cluster offset in Lab
	convergenceEps   = 1e-3 // palette movement that counts as converged
	maxIterations    = 1000
)

// Resize downscales src to width x height and quantizes it to at most colors
// colors, jointly. Width and height must be positive and no larger than src;
// colors must be at least 1. src is read as sRGB-encoded straight color;
// alpha is ignored and the palette is opaque.
func Resize(src *pix.Image, width, height, colors int) (*image.Paletted, error) {
	if src == nil || src.W == 0 || src.H == 0 {
		return nil, errors.New("pixelate: empty image")
	}
	if width <= 0 || height <= 0 || width > src.W || height > src.H {
		return nil, errors.New("pixelate: output dimensions must be positive and no larger than the input")
	}
	if colors < 1 {
		return nil, errors.New("pixelate: need at least one color")
	}

	in := readLab(src)
	iw, ih := src.W, src.H
	n := width * height

	// Superpixels: one per output pixel, initialized on a regular grid.
	sp := make([]superpixel, n)
	sx, sy := float64(iw)/float64(width), float64(ih)/float64(height)
	for y := range height {
		for x := range width {
			sp[x+y*width].x = (float64(x) + 0.5) * sx
			sp[x+y*width].y = (float64(y) + 0.5) * sy
		}
	}
	assign := make([]int, iw*ih)
	assignSuperpixels(in, iw, ih, sp, width, height, assign, true)

	// Palette: starts as one pair of sub-clusters at the mean superpixel color,
	// temperature from the color covariance's largest eigenvalue.
	mean := chroma.Lab{}
	for _, s := range sp {
		mean.L += s.color.L / float64(n)
		mean.A += s.color.A / float64(n)
		mean.B += s.color.B / float64(n)
	}
	temp := initialTempScale * 2 * largestEigenvalue(sp, mean)
	pal := []cluster{{mean, 0.5}, {mean, 0.5}}
	pairs := [][2]int{{0, 1}}
	perturb(pal, pairs)

	prob := make([]float64, 0) // prob[s*len(pal)+k] = P(k | s)
	// Superpixels settle within a few iterations; after that the SLIC step
	// over the whole source is only worth repeating when the temperature
	// changes.
	refineSuperpixels := true
	for iter := 0; iter < maxIterations; iter++ {
		if refineSuperpixels || iter < 10 {
			assignSuperpixels(in, iw, ih, sp, width, height, assign, false)
			smoothColors(sp, width, height)
			refineSuperpixels = false
		}

		prob = associate(sp, pal, temp, prob)
		moved := refinePalette(sp, pal, prob)

		if moved < convergenceEps {
			if temp <= finalTemp {
				break
			}
			temp = math.Max(temp*coolingRate, finalTemp)
			refineSuperpixels = true
			if len(pairs) < colors {
				pal, pairs = expand(pal, pairs, colors)
			}
		}
	}
	// Each superpixel takes the most likely palette color.
	dst := image.NewPaletted(image.Rect(0, 0, width, height), make(color.Palette, 0, len(pairs)))
	pairColor := make([]int, len(pal))
	for i, pr := range pairs {
		c := pal[pr[0]].c
		if pal[pr[1]].p > 0 || pal[pr[0]].p > 0 {
			w := pal[pr[0]].p + pal[pr[1]].p
			c = chroma.Lab{
				L: (pal[pr[0]].c.L*pal[pr[0]].p + pal[pr[1]].c.L*pal[pr[1]].p) / w,
				A: (pal[pr[0]].c.A*pal[pr[0]].p + pal[pr[1]].c.A*pal[pr[1]].p) / w,
				B: (pal[pr[0]].c.B*pal[pr[0]].p + pal[pr[1]].c.B*pal[pr[1]].p) / w,
			}
		}
		r, g, b := chroma.SRGBFromRGB(chroma.RGBFromXYZ(c.XYZ())).Clamp().To8()
		dst.Palette = append(dst.Palette, color.NRGBA{r, g, b, 255})
		pairColor[pr[0]], pairColor[pr[1]] = i, i
	}
	k := len(pal)
	for s := range sp {
		best := 0
		for j := 1; j < k; j++ {
			if prob[s*k+j] > prob[s*k+best] {
				best = j
			}
		}
		dst.Pix[s] = uint8(pairColor[best])
	}
	return dst, nil
}

type superpixel struct {
	x, y   float64
	color  chroma.Lab // mean color of members
	smooth chroma.Lab // bilateral-smoothed color used for palette association
	count  int
}

type cluster struct {
	c chroma.Lab
	p float64 // P(k)
}

func readLab(m *pix.Image) []chroma.Lab {
	out := make([]chroma.Lab, m.W*m.H)
	for i := range out {
		r, g, b, a := m.Pix[4*i], m.Pix[4*i+1], m.Pix[4*i+2], m.Pix[4*i+3]
		if a > 0 {
			r, g, b = r/a, g/a, b/a
		}
		out[i] = chroma.LabFromXYZ(chroma.SRGB{R: r, G: g, B: b}.RGB().XYZ())
	}
	return out
}

// assignSuperpixels runs one SLIC step: every input pixel joins the nearest
// of its 3x3 neighboring superpixels, then superpixels move to their members'
// mean position and color. When init is set, colors come from the grid
// position only.
func assignSuperpixels(in []chroma.Lab, iw, ih int, sp []superpixel, w, h int, assign []int, init bool) {
	sx, sy := float64(iw)/float64(w), float64(ih)/float64(h)
	if init {
		for i := range sp {
			x, y := min(int(sp[i].x), iw-1), min(int(sp[i].y), ih-1)
			sp[i].color = in[x+y*iw]
		}
	}
	// Normalize position distance by the superpixel spacing (paper: sqrt(N/K)).
	scale := spatialWeight / math.Sqrt(sx*sy)
	for py := range ih {
		for px := range iw {
			cx, cy := int(float64(px)/sx), int(float64(py)/sy)
			best, bestD := -1, math.Inf(1)
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					x, y := cx+dx, cy+dy
					if x < 0 || y < 0 || x >= w || y >= h {
						continue
					}
					s := &sp[x+y*w]
					ddx, ddy := float64(px)+0.5-s.x, float64(py)+0.5-s.y
					d := in[px+py*iw].Distance(s.color) + scale*math.Sqrt(ddx*ddx+ddy*ddy)
					if d < bestD {
						best, bestD = x+y*w, d
					}
				}
			}
			assign[px+py*iw] = best
		}
	}
	for i := range sp {
		sp[i].x, sp[i].y, sp[i].color, sp[i].count = 0, 0, chroma.Lab{}, 0
	}
	for py := range ih {
		for px := range iw {
			s := &sp[assign[px+py*iw]]
			c := in[px+py*iw]
			s.x += float64(px) + 0.5
			s.y += float64(py) + 0.5
			s.color.L += c.L
			s.color.A += c.A
			s.color.B += c.B
			s.count++
		}
	}
	for i := range sp {
		s := &sp[i]
		if s.count == 0 {
			// Empty superpixel: reset to its grid cell so it can recapture pixels.
			s.x = (float64(i%w) + 0.5) * sx
			s.y = (float64(i/w) + 0.5) * sy
			x, y := min(int(s.x), iw-1), min(int(s.y), ih-1)
			s.color = in[x+y*iw]
			continue
		}
		n := float64(s.count)
		s.x, s.y = s.x/n, s.y/n
		s.color = chroma.Lab{L: s.color.L / n, A: s.color.A / n, B: s.color.B / n}
	}
	// Laplacian smoothing of positions keeps the grid regular (paper 4.1).
	for y := range h {
		for x := range w {
			var ax, ay float64
			cnt := 0
			for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				ax += sp[nx+ny*w].x
				ay += sp[nx+ny*w].y
				cnt++
			}
			if cnt > 0 {
				s := &sp[x+y*w]
				s.x = 0.6*s.x + 0.4*ax/float64(cnt)
				s.y = 0.6*s.y + 0.4*ay/float64(cnt)
			}
		}
	}
}

// smoothColors sets each superpixel's smooth color to a 3x3 bilateral blend
// of its neighbors' mean colors (paper 4.1), so palette association sees
// less noise than the raw means.
func smoothColors(sp []superpixel, w, h int) {
	const sigma = 5.0 // Lab units
	for y := range h {
		for x := range w {
			c := sp[x+y*w].color
			var sum chroma.Lab
			var wsum float64
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					o := sp[nx+ny*w].color
					d := c.Distance(o)
					wt := math.Exp(-d * d / (2 * sigma * sigma))
					sum.L += o.L * wt
					sum.A += o.A * wt
					sum.B += o.B * wt
					wsum += wt
				}
			}
			sp[x+y*w].smooth = chroma.Lab{L: sum.L / wsum, A: sum.A / wsum, B: sum.B / wsum}
		}
	}
}

// associate computes P(k | s) ∝ P(k) exp(-d(s,k)²/T) for every superpixel s
// and palette cluster k, returning the table indexed by s*len(pal)+k.
func associate(sp []superpixel, pal []cluster, temp float64, prob []float64) []float64 {
	k := len(pal)
	prob = prob[:0]
	for _, s := range sp {
		start := len(prob)
		var total float64
		minD := math.Inf(1)
		for _, c := range pal {
			minD = math.Min(minD, sq(s.smooth.Distance(c.c)))
		}
		for _, c := range pal {
			p := c.p * math.Exp(-(sq(s.smooth.Distance(c.c))-minD)/temp)
			prob = append(prob, p)
			total += p
		}
		for i := start; i < start+k; i++ {
			prob[i] /= total
		}
	}
	return prob
}

// refinePalette moves each cluster to the probability-weighted mean of the
// superpixels and updates P(k). Returns the largest movement in Lab.
func refinePalette(sp []superpixel, pal []cluster, prob []float64) float64 {
	k := len(pal)
	ps := 1 / float64(len(sp))
	moved := 0.0
	for j := range pal {
		var sum chroma.Lab
		var pk float64
		for s := range sp {
			p := prob[s*k+j] * ps
			sum.L += sp[s].smooth.L * p
			sum.A += sp[s].smooth.A * p
			sum.B += sp[s].smooth.B * p
			pk += p
		}
		if pk <= 0 {
			pal[j].p = 0
			continue
		}
		next := chroma.Lab{L: sum.L / pk, A: sum.A / pk, B: sum.B / pk}
		moved = math.Max(moved, next.Distance(pal[j].c))
		pal[j] = cluster{next, pk}
	}
	return moved
}

// expand splits every pair whose sub-clusters drifted apart into two pairs,
// until there are limit pairs, then perturbs coincident sub-clusters so they
// can drift again.
func expand(pal []cluster, pairs [][2]int, limit int) ([]cluster, [][2]int) {
	n := len(pairs)
	for i := 0; i < n && len(pairs) < limit; i++ {
		a, b := pairs[i][0], pairs[i][1]
		if pal[a].c.Distance(pal[b].c) <= splitDistance {
			continue
		}
		pal[a].p /= 2
		pal = append(pal, pal[a])
		pairs[i] = [2]int{a, len(pal) - 1}
		pal[b].p /= 2
		pal = append(pal, pal[b])
		pairs = append(pairs, [2]int{b, len(pal) - 1})
	}
	perturb(pal, pairs)
	return pal, pairs
}

// perturb pushes the two sub-clusters of each pair apart by perturbation
// along their current axis (or along A when they coincide) so annealing
// can tell them apart.
func perturb(pal []cluster, pairs [][2]int) {
	for _, pr := range pairs {
		a, b := &pal[pr[0]], &pal[pr[1]]
		d := b.c.Distance(a.c)
		if d >= splitDistance {
			continue
		}
		dir := chroma.Lab{A: 1}
		if d > 1e-9 {
			dir = chroma.Lab{L: (b.c.L - a.c.L) / d, A: (b.c.A - a.c.A) / d, B: (b.c.B - a.c.B) / d}
		}
		a.c.L -= dir.L * perturbation
		a.c.A -= dir.A * perturbation
		a.c.B -= dir.B * perturbation
		b.c.L += dir.L * perturbation
		b.c.A += dir.A * perturbation
		b.c.B += dir.B * perturbation
	}
}

// largestEigenvalue returns the largest eigenvalue of the superpixel colors'
// covariance matrix, via power iteration.
func largestEigenvalue(sp []superpixel, mean chroma.Lab) float64 {
	var cov [3][3]float64
	for _, s := range sp {
		d := [3]float64{s.color.L - mean.L, s.color.A - mean.A, s.color.B - mean.B}
		for i := range 3 {
			for j := range 3 {
				cov[i][j] += d[i] * d[j] / float64(len(sp))
			}
		}
	}
	v := [3]float64{1, 1, 1}
	var lambda float64
	for range 50 {
		var nv [3]float64
		for i := range 3 {
			for j := range 3 {
				nv[i] += cov[i][j] * v[j]
			}
		}
		lambda = math.Sqrt(nv[0]*nv[0] + nv[1]*nv[1] + nv[2]*nv[2])
		if lambda == 0 {
			break
		}
		for i := range 3 {
			v[i] = nv[i] / lambda
		}
	}
	return math.Max(lambda, 1e-6)
}

func sq(x float64) float64 { return x * x }
