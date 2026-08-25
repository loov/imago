package chroma

import (
	"math"
	"testing"
)

var grid = func() []SRGB {
	cs := []SRGB{
		{0, 0, 0}, {1, 1, 1}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		{1, 1, 0}, {0, 1, 1}, {1, 0, 1}, {0.5, 0.5, 0.5}, {0.01, 0.02, 0.03},
	}
	for i := 0; i < 20; i++ {
		f := float64(i)
		cs = append(cs, SRGB{
			math.Mod(f*0.37+0.11, 1),
			math.Mod(f*0.53+0.29, 1),
			math.Mod(f*0.71+0.07, 1),
		})
	}
	return cs
}()

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > tol {
		t.Errorf("%s: got %v want %v", name, got, want)
	}
}

func nearHue(t *testing.T, name string, gotH, wantH, c float64) {
	t.Helper()
	if c < 1e-9 {
		return
	}
	d := math.Abs(gotH - wantH)
	if math.IsNaN(d) || math.IsInf(d, 0) {
		t.Errorf("%s: got %v want %v", name, gotH, wantH)
		return
	}
	if d > 0.5 {
		d = 1 - d
	}
	if d > 1e-6 {
		t.Errorf("%s: got %v want %v", name, gotH, wantH)
	}
}

func TestRoundtrip(t *testing.T) {
	for _, s := range grid {
		rgb := s.RGB()
		xyz := rgb.XYZ()

		s2 := SRGBFromRGB(rgb)
		near(t, "srgb.R", s2.R, s.R, 1e-9)
		near(t, "srgb.G", s2.G, s.G, 1e-9)
		near(t, "srgb.B", s2.B, s.B, 1e-9)

		r2 := RGBFromXYZ(xyz)
		near(t, "xyz.R", r2.R, rgb.R, 1e-9)
		near(t, "xyz.G", r2.G, rgb.G, 1e-9)
		near(t, "xyz.B", r2.B, rgb.B, 1e-9)

		ok := OklabFromRGB(rgb)
		r3 := ok.RGB()
		near(t, "oklab.R", r3.R, rgb.R, 1e-9)
		near(t, "oklab.G", r3.G, rgb.G, 1e-9)
		near(t, "oklab.B", r3.B, rgb.B, 1e-9)

		olch := OkLChFromOklab(ok)
		ok2 := olch.Oklab()
		near(t, "oklch.L", ok2.L, ok.L, 1e-9)
		near(t, "oklch.A", ok2.A, ok.A, 1e-9)
		near(t, "oklch.B", ok2.B, ok.B, 1e-9)
		nearHue(t, "oklch.H", OkLChFromOklab(ok2).H, olch.H, olch.C)

		lab := LabFromXYZ(xyz)
		x2 := lab.XYZ()
		near(t, "lab.X", x2.X, xyz.X, 1e-9)
		near(t, "lab.Y", x2.Y, xyz.Y, 1e-9)
		near(t, "lab.Z", x2.Z, xyz.Z, 1e-9)

		lch := LChFromLab(lab)
		lab2 := lch.Lab()
		near(t, "lch.L", lab2.L, lab.L, 1e-9)
		near(t, "lch.A", lab2.A, lab.A, 1e-9)
		near(t, "lch.B", lab2.B, lab.B, 1e-9)
		nearHue(t, "lch.H", LChFromLab(lab2).H, lch.H, lch.C)

		luv := LuvFromXYZ(xyz)
		x3 := luv.XYZ()
		near(t, "luv.X", x3.X, xyz.X, 1e-9)
		near(t, "luv.Y", x3.Y, xyz.Y, 1e-9)
		near(t, "luv.Z", x3.Z, xyz.Z, 1e-9)

		hsl := HSLFromSRGB(s)
		s3 := hsl.SRGB()
		near(t, "hsl.R", s3.R, s.R, 1e-9)
		near(t, "hsl.G", s3.G, s.G, 1e-9)
		near(t, "hsl.B", s3.B, s.B, 1e-9)

		hsv := HSVFromSRGB(s)
		s4 := hsv.SRGB()
		near(t, "hsv.R", s4.R, s.R, 1e-9)
		near(t, "hsv.G", s4.G, s.G, 1e-9)
		near(t, "hsv.B", s4.B, s.B, 1e-9)
	}
}
