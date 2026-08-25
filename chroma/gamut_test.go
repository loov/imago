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
