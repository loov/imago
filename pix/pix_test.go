package pix

import (
	"image"
	"image/color"
	"math/rand/v2"
	"testing"
)

// hidden hides the concrete type so FromImage takes the generic path.
// The generic path routes through RGBA→NRGBA64, which loses sub-byte precision,
// so comparisons allow half an 8-bit step.
type hidden struct{ image.Image }

func sample(r image.Rectangle, seed uint64) *image.NRGBA {
	rng := rand.New(rand.NewPCG(seed, 0))
	m := image.NewNRGBA(r)
	for i := range m.Pix {
		m.Pix[i] = uint8(rng.IntN(256))
	}
	return m
}

func TestFromImageBounds(t *testing.T) {
	src := sample(image.Rect(3, 5, 10, 9), 1)
	m := FromImage(src)
	if m.W != 7 || m.H != 4 {
		t.Fatalf("size %dx%d", m.W, m.H)
	}
	c := src.NRGBAAt(5, 7)
	r, g, b, a := m.At(2, 2)
	fa := float64(c.A) / 255
	if a != fa || r != float64(c.R)/255*fa || g != float64(c.G)/255*fa || b != float64(c.B)/255*fa {
		t.Fatalf("pixel mismatch: %v vs %v %v %v %v", c, r, g, b, a)
	}
	if e := FromImage(image.NewNRGBA(image.Rect(0, 0, 0, 5))); e.W != 0 || e.Pix != nil {
		t.Fatalf("empty bounds: %+v", e)
	}
}

func TestFastPathMatchesGeneric(t *testing.T) {
	src := sample(image.Rect(2, 2, 9, 8), 2)
	fast, slow := FromImage(src), FromImage(hidden{src})
	for i := range fast.Pix {
		if d := fast.Pix[i] - slow.Pix[i]; d > 0.5/255 || d < -0.5/255 {
			t.Fatalf("pix[%d]: %v vs %v", i, fast.Pix[i], slow.Pix[i])
		}
	}
	rgba := image.NewRGBA(src.Bounds())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			rgba.Set(x, y, src.At(x, y))
		}
	}
	fast, slow = FromImage(rgba), FromImage(hidden{rgba})
	for i := range fast.Pix {
		if d := fast.Pix[i] - slow.Pix[i]; d > 0.5/255 || d < -0.5/255 {
			t.Fatalf("rgba pix[%d]: %v vs %v", i, fast.Pix[i], slow.Pix[i])
		}
	}
}

func TestRoundTrip(t *testing.T) {
	src := sample(image.Rect(0, 0, 64, 64), 3)
	for i := 3; i < len(src.Pix); i += 4 {
		src.Pix[i] = max(src.Pix[i], 1)
	}
	for x := range 64 {
		src.SetNRGBA(x, 0, color.NRGBA{uint8(x * 4), 200, 17, 255})
	}
	out := FromImage(src).NRGBA()
	for i := range src.Pix {
		a := src.Pix[i|3]
		d := int(out.Pix[i]) - int(src.Pix[i])
		if d < 0 {
			d = -d
		}
		if d > 1 || (a == 255 && d != 0) {
			t.Fatalf("pix[%d]: got %d want %d (alpha %d)", i, out.Pix[i], src.Pix[i], a)
		}
	}
	src.SetNRGBA(1, 1, color.NRGBA{10, 20, 30, 0})
	if c := FromImage(src).NRGBA().NRGBAAt(1, 1); c != (color.NRGBA{}) {
		t.Fatalf("alpha 0: %v", c)
	}
}

func TestClone(t *testing.T) {
	m := FromImage(sample(image.Rect(4, 6, 12, 11), 4))
	c := m.Clone()
	c.Pix[0]++
	if m.Pix[0] == c.Pix[0] {
		t.Fatal("Clone shares Pix")
	}
}

func TestChannel(t *testing.T) {
	m := FromImage(sample(image.Rect(0, 0, 5, 3), 5))
	orig := m.Clone()
	for i := range 4 {
		m.SetChannel(i, m.Channel(i))
	}
	for i := range m.Pix {
		if m.Pix[i] != orig.Pix[i] {
			t.Fatalf("pix[%d] changed", i)
		}
	}
}

func TestLinearize(t *testing.T) {
	m := FromImage(sample(image.Rect(0, 0, 16, 16), 6))
	back := m.Linearize().Delinearize()
	for i := range m.Pix {
		if d := back.Pix[i] - m.Pix[i]; d > 1e-9 || d < -1e-9 {
			t.Fatalf("pix[%d]: %v vs %v", i, back.Pix[i], m.Pix[i])
		}
	}
	lin := (&Image{W: 2, H: 1, Pix: []float64{0, 0, 0, 1, 1, 1, 1, 1}}).Linearize()
	avg := New(1, 1)
	for c := range 4 {
		avg.Pix[c] = (lin.Pix[c] + lin.Pix[4+c]) / 2
	}
	if v := avg.Delinearize().Pix[0]; v < 0.734 || v > 0.737 {
		t.Fatalf("linear average of 0 and 1 = %v, want ~0.735", v)
	}
	if z := (&Image{W: 1, H: 1, Pix: []float64{0, 0, 0, 0}}).Linearize().Pix; z[0] != 0 {
		t.Fatalf("alpha 0: %v", z)
	}
}

func mustPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s did not panic", name)
		}
	}()
	f()
}

func TestClamp(t *testing.T) {
	m := New(1, 1)
	m.Set(0, 0, 0.9, -0.1, 0.2, 0.5)
	m.Clamp()
	if r, g, b, a := m.At(0, 0); r != 0.5 || g != 0 || b != 0.2 || a != 0.5 {
		t.Fatalf("got %v %v %v %v", r, g, b, a)
	}
	m.Set(0, 0, 2, 0, 0, 1.5)
	if r, _, _, a := m.Clamp().At(0, 0); r != 1 || a != 1 {
		t.Fatalf("got %v %v", r, a)
	}
}

func TestValidation(t *testing.T) {
	m := New(2, 2)
	mustPanic(t, "At(2,0)", func() { m.At(2, 0) })
	mustPanic(t, "At(0,2)", func() { m.At(0, 2) })
	mustPanic(t, "At(-1,0)", func() { m.At(-1, 0) })
	mustPanic(t, "Set(0,-1)", func() { m.Set(0, -1, 0, 0, 0, 0) })
	mustPanic(t, "SetChannel(4)", func() { m.SetChannel(4, make([]float64, 4)) })
	mustPanic(t, "SetChannel short", func() { m.SetChannel(0, make([]float64, 3)) })
	mustPanic(t, "SetChannel long", func() { m.SetChannel(0, make([]float64, 5)) })
	for _, v := range m.Pix {
		if v != 0 {
			t.Fatal("SetChannel wrote before validating")
		}
	}
	mustPanic(t, "New(-1,1)", func() { New(-1, 1) })
	mustPanic(t, "New(1,-1)", func() { New(1, -1) })
	mustPanic(t, "New overflow", func() { New(1<<40, 1<<40) })
	if e := New(0, 5); e.Pix != nil || e.H != 5 {
		t.Fatalf("New(0,5) = %+v", e)
	}
}
