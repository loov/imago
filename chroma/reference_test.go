package chroma

import "testing"

func TestOklabReference(t *testing.T) {
	w := OklabFromRGB(SRGB{1, 1, 1}.RGB())
	near(t, "white.L", w.L, 1, 1e-3)
	near(t, "white.A", w.A, 0, 1e-3)
	near(t, "white.B", w.B, 0, 1e-3)

	// Ottosson's published XYZ -> Oklab vectors.
	vectors := []struct {
		in   XYZ
		want Oklab
	}{
		{XYZ{0.950, 1.000, 1.089}, Oklab{1.000, 0.000, 0.000}},
		{XYZ{1, 0, 0}, Oklab{0.450, 1.236, -0.019}},
		{XYZ{0, 1, 0}, Oklab{0.922, -0.671, 0.263}},
		{XYZ{0, 0, 1}, Oklab{0.153, -1.415, -0.449}},
	}
	for _, v := range vectors {
		got := OklabFromRGB(RGBFromXYZ(v.in))
		near(t, "L", got.L, v.want.L, 2e-3)
		near(t, "A", got.A, v.want.A, 2e-3)
		near(t, "B", got.B, v.want.B, 2e-3)
	}
}

func TestLabLuvReference(t *testing.T) {
	w := LabFromXYZ(SRGB{1, 1, 1}.RGB().XYZ())
	near(t, "white.L", w.L, 100, 0.05)
	near(t, "white.A", w.A, 0, 0.05)
	near(t, "white.B", w.B, 0, 0.05)

	red := SRGB{1, 0, 0}.RGB().XYZ()
	lab := LabFromXYZ(red)
	near(t, "red.L", lab.L, 53.24, 0.05)
	near(t, "red.A", lab.A, 80.09, 0.05)
	near(t, "red.B", lab.B, 67.20, 0.05)

	luv := LuvFromXYZ(red)
	near(t, "red.L", luv.L, 53.24, 0.1)
	near(t, "red.U", luv.U, 175.02, 0.1)
	near(t, "red.V", luv.V, 37.76, 0.1)
}

func TestHSLHSVReference(t *testing.T) {
	cases := []struct {
		in  SRGB
		hsl HSL
		hsv HSV
	}{
		{SRGB{1, 0, 0}, HSL{0, 1, 0.5}, HSV{0, 1, 1}},
		{SRGB{0, 1, 0}, HSL{1.0 / 3, 1, 0.5}, HSV{1.0 / 3, 1, 1}},
		{SRGB{0, 0, 1}, HSL{2.0 / 3, 1, 0.5}, HSV{2.0 / 3, 1, 1}},
		{SRGB{1, 1, 1}, HSL{0, 0, 1}, HSV{0, 0, 1}},
		{SRGB{0, 0, 0}, HSL{0, 0, 0}, HSV{0, 0, 0}},
	}
	for _, c := range cases {
		if got := HSLFromSRGB(c.in); got != c.hsl {
			t.Errorf("HSL(%v) = %v want %v", c.in, got, c.hsl)
		}
		if got := HSVFromSRGB(c.in); got != c.hsv {
			t.Errorf("HSV(%v) = %v want %v", c.in, got, c.hsv)
		}
	}
}

func TestTransfer(t *testing.T) {
	near(t, "ToLinear(0.5)", ToLinear(0.5), 0.214041, 1e-6)
	near(t, "ToSRGB", ToSRGB(ToLinear(0.5)), 0.5, 1e-12)
}

func TestSRGB8RoundTrip(t *testing.T) {
	for _, v := range []uint8{0, 1, 127, 128, 254, 255} {
		r, g, b := SRGBFrom8(v, 255-v, 7).To8()
		if r != v || g != 255-v || b != 7 {
			t.Fatalf("%d: got %d %d %d", v, r, g, b)
		}
	}
	if d := (Oklab{1, 0, 0}).Distance(Oklab{0, 0, 0}); d != 1 {
		t.Fatalf("Distance = %v, want 1", d)
	}
}
