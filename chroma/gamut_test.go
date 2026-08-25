package chroma

import "testing"

func TestClampDistance(t *testing.T) {
	if got := (RGB{-1, 0.5, 2}).Clamp(); got != (RGB{0, 0.5, 1}) {
		t.Errorf("RGB.Clamp = %v", got)
	}
	near(t, "ΔE76", (Lab{L: 100}).Distance(Lab{}), 100, 1e-12)

	in := OkLChFromOklab(OklabFromRGB(SRGB{0.2, 0.5, 0.7}.RGB()))
	if got := in.Clamp(); got != in {
		t.Errorf("Clamp in-gamut = %v want %v", got, in)
	}
	out := OkLCh{L: 0.7, C: 0.4, H: 0.3}
	got := out.Clamp()
	if got.L != out.L || got.H != out.H || got.C >= out.C {
		t.Errorf("Clamp = %v", got)
	}
	s := SRGBFromRGB(got.Oklab().RGB().Clamp())
	r := got.Oklab().RGB()
	for _, v := range []float64{r.R, r.G, r.B} {
		if v < -1e-3 || v > 1+1e-3 {
			t.Errorf("out of gamut: %v (%v)", r, s)
		}
	}
}

func TestLuvBlack(t *testing.T) {
	l := LuvFromXYZ(XYZ{})
	near(t, "L", l.L, 0, 0)
	near(t, "U", l.U, 0, 0)
	near(t, "V", l.V, 0, 0)
	x := l.XYZ()
	near(t, "X", x.X, 0, 0)
	near(t, "Y", x.Y, 0, 0)
	near(t, "Z", x.Z, 0, 0)
}

func TestOkLChClampL(t *testing.T) {
	c := OkLCh{L: 1.05, C: 0.1, H: 0.5}.Clamp()
	near(t, "L", c.L, 1, 0)
	near(t, "C", c.C, 0, 1e-9)
}
