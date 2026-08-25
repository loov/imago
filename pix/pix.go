// Package pix provides a premultiplied float RGBA image used as the common
// pixel I/O layer for scaling algorithms.
package pix

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/loov/imago/chroma"
)

// Image is a premultiplied RGBA float image with origin (0,0).
// Values are in the encoding of the source (sRGB-encoded for sRGB input); Image does no color conversion.
//
// Invariant: A is in [0,1] and each of R, G, B is in [0,A]. Filters may
// temporarily violate this; Clamp restores it.
type Image struct {
	W, H int
	Pix  []float64 // 4*W*H, premultiplied R,G,B,A in 0..1, row-major
}

// New returns a zeroed image of the given size.
func New(w, h int) *Image {
	if w < 0 || h < 0 {
		panic("pix: negative image dimensions")
	}
	if w != 0 && h > math.MaxInt/(4*w) {
		panic("pix: image dimensions overflow")
	}
	if w == 0 || h == 0 {
		return &Image{W: w, H: h}
	}
	return &Image{W: w, H: h, Pix: make([]float64, 4*w*h)}
}

// FromImage reads src into a premultiplied float image with origin (0,0).
func FromImage(src image.Image) *Image {
	b := src.Bounds()
	m := New(b.Dx(), b.Dy())
	if m.Pix == nil {
		return m
	}
	switch s := src.(type) {
	case *image.NRGBA:
		for y := range m.H {
			row := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+y):]
			for x := range m.W {
				a := float64(row[4*x+3]) / 255
				i := 4 * (x + y*m.W)
				m.Pix[i+0] = float64(row[4*x+0]) / 255 * a
				m.Pix[i+1] = float64(row[4*x+1]) / 255 * a
				m.Pix[i+2] = float64(row[4*x+2]) / 255 * a
				m.Pix[i+3] = a
			}
		}
	case *image.RGBA:
		for y := range m.H {
			row := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+y):]
			for x := range m.W {
				i := 4 * (x + y*m.W)
				for c := range 4 {
					m.Pix[i+c] = float64(row[4*x+c]) / 255
				}
			}
		}
	default:
		for y := range m.H {
			for x := range m.W {
				c := color.NRGBA64Model.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA64)
				a := float64(c.A) / 65535
				m.Set(x, y, float64(c.R)/65535*a, float64(c.G)/65535*a, float64(c.B)/65535*a, a)
			}
		}
	}
	return m
}

// At returns the premultiplied components at (x, y).
func (m *Image) At(x, y int) (r, g, b, a float64) {
	i := m.offset(x, y)
	return m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3]
}

// Set stores premultiplied components at (x, y).
func (m *Image) Set(x, y int, r, g, b, a float64) {
	i := m.offset(x, y)
	m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3] = r, g, b, a
}

func (m *Image) offset(x, y int) int {
	if x < 0 || x >= m.W || y < 0 || y >= m.H {
		panic(fmt.Sprintf("pix: (%d,%d) out of bounds of %dx%d image", x, y, m.W, m.H))
	}
	return 4 * (x + y*m.W)
}

// Clamp restores the invariant in place and returns m:
// A ← clamp(A,0,1); R,G,B ← clamp(v,0,A).
func (m *Image) Clamp() *Image {
	for p := 0; p < len(m.Pix); p += 4 {
		a := min(max(m.Pix[p+3], 0), 1)
		m.Pix[p+3] = a
		for c := range 3 {
			m.Pix[p+c] = min(max(m.Pix[p+c], 0), a)
		}
	}
	return m
}

// Clone returns a deep copy.
func (m *Image) Clone() *Image {
	c := *m
	if m.Pix != nil {
		c.Pix = append([]float64(nil), m.Pix...)
	}
	return &c
}

// Channel copies channel i (0..3) into a new slice of W*H values.
func (m *Image) Channel(i int) []float64 {
	v := make([]float64, m.W*m.H)
	for p := range v {
		v[p] = m.Pix[4*p+i]
	}
	return v
}

// SetChannel writes W*H values into channel i (0..3).
func (m *Image) SetChannel(i int, v []float64) {
	if i < 0 || i > 3 {
		panic(fmt.Sprintf("pix: channel %d out of range", i))
	}
	if len(v) != m.W*m.H {
		panic(fmt.Sprintf("pix: SetChannel got %d values, want %d", len(v), m.W*m.H))
	}
	for p, x := range v {
		m.Pix[4*p+i] = x
	}
}

// NRGBA un-premultiplies (alpha 0 yields RGB 0), clamps to 0..1 and rounds to bytes.
func (m *Image) NRGBA() *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, m.W, m.H))
	for p := range m.W * m.H {
		a := m.Pix[4*p+3]
		if a > 0 {
			dst.Pix[4*p+0] = toByte(m.Pix[4*p+0] / a)
			dst.Pix[4*p+1] = toByte(m.Pix[4*p+1] / a)
			dst.Pix[4*p+2] = toByte(m.Pix[4*p+2] / a)
		}
		dst.Pix[4*p+3] = toByte(a)
	}
	return dst
}

func toByte(v float64) uint8 {
	return uint8(math.Round(min(max(v, 0), 1) * 255))
}

// Linearize returns a copy with sRGB-encoded straight color converted to linear light (alpha unchanged, premultiplication preserved).
func (m *Image) Linearize() *Image { return mapColor(m, chroma.ToLinear) }

// Delinearize is the inverse of Linearize.
func (m *Image) Delinearize() *Image { return mapColor(m, chroma.ToSRGB) }

// mapColor applies f to the straight (un-premultiplied) color of every pixel of a copy.
func mapColor(m *Image, f func(float64) float64) *Image {
	c := m.Clone()
	for p := range c.W * c.H {
		a := c.Pix[4*p+3]
		if a <= 0 {
			continue
		}
		for i := range 3 {
			c.Pix[4*p+i] = f(c.Pix[4*p+i]/a) * a
		}
	}
	return c
}
