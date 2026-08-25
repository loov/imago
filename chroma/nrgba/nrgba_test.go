package nrgba

import (
	"image/color"
	"testing"

	"github.com/loov/imago/chroma"
)

func TestPremultiply(t *testing.T) {
	if got, want := Premultiply(color.NRGBA{255, 255, 255, 128}), (color.RGBA{128, 128, 128, 128}); got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	for col := 0; col <= 0xFF; col += 5 {
		for alpha := 0; alpha <= 0xFF; alpha += 5 {
			c := color.NRGBA{R: uint8(col), G: uint8(255 - col), B: uint8(col / 2), A: uint8(alpha)}
			p := Premultiply(c)
			if p.R > p.A || p.G > p.A || p.B > p.A {
				t.Fatalf("%v: invariant broken %v", c, p)
			}
			if want := color.RGBAModel.Convert(c).(color.RGBA); p != want {
				t.Fatalf("%v: got %v want %v", c, p, want)
			}
			u := Unpremultiply(p)
			if want := color.NRGBAModel.Convert(p).(color.NRGBA); u != want {
				t.Fatalf("%v: unpremultiply got %v want %v", p, u, want)
			}
			if alpha == 0xFF && u != c {
				t.Fatalf("%v: roundtrip got %v", c, u)
			}
			if alpha >= 128 {
				// 8-bit premultiplication loses 255/alpha of precision.
				tol := (255 + alpha - 1) / alpha
				for _, d := range [][2]uint8{{u.R, c.R}, {u.G, c.G}, {u.B, c.B}} {
					if x, y := int(d[0]), int(d[1]); x-y > tol || y-x > tol {
						t.Fatalf("%v: roundtrip got %v", c, u)
					}
				}
			}
		}
	}
}

func TestLinearRoundtrip(t *testing.T) {
	for col := 0; col <= 0xFF; col++ {
		for alpha := 0; alpha <= 0xFF; alpha++ {
			want := color.NRGBA{R: uint8(col), A: uint8(alpha)}
			if alpha == 0 {
				want.R = 0
			}
			if got := ToLinear(want).NRGBA(); want != got {
				t.Fatalf("got %v expected %v", got, want)
			}
		}
	}
}

func TestNRGBA(t *testing.T) {
	cases := []struct {
		in   chroma.HSL
		want color.NRGBA
	}{
		{chroma.HSL{H: 0, S: 1, L: 0.5}, color.NRGBA{255, 0, 0, 255}},
		{chroma.HSL{H: 1.0 / 3, S: 1, L: 0.5}, color.NRGBA{0, 255, 0, 255}},
		{chroma.HSL{H: 2.0 / 3, S: 1, L: 0.5}, color.NRGBA{0, 0, 255, 255}},
		{chroma.HSL{H: 0, S: 0, L: 0.5}, color.NRGBA{128, 128, 128, 255}},
	}
	for _, c := range cases {
		if got := FromSRGB(c.in.SRGB()); got != c.want {
			t.Errorf("%v: got %v want %v", c.in, got, c.want)
		}
	}
	if got := SRGB(Hex(0x11223344)); got != chroma.SRGBFrom8(0x11, 0x22, 0x33) {
		t.Errorf("SRGB = %v", got)
	}
}

func TestBlend(t *testing.T) {
	if got, want := Hex(0x11223344), (color.NRGBA{0x11, 0x22, 0x33, 0x44}); got != want {
		t.Errorf("Hex: got %v want %v", got, want)
	}
	a, b := color.NRGBA{10, 20, 30, 255}, color.NRGBA{200, 100, 50, 255}
	if got := Lerp(a, b, 0); got != a {
		t.Errorf("Lerp p=0: got %v want %v", got, a)
	}
	if got := Lerp(a, b, 1); got != b {
		t.Errorf("Lerp p=1: got %v want %v", got, b)
	}
	if got := Lerp(a, b, 0.5); got != (color.NRGBA{105, 60, 40, 255}) {
		t.Errorf("Lerp p=0.5: got %v", got)
	}
	c := color.NRGBA{10, 20, 30, 200}
	if got := MulAlpha(c, 255); got != c {
		t.Errorf("MulAlpha 255: got %v want %v", got, c)
	}
	if got := MulAlpha(c, 0); got.A != 0 {
		t.Errorf("MulAlpha 0: got %v", got)
	}
	for col := 0; col <= 0xFF; col++ {
		c := color.NRGBA{R: uint8(col), G: uint8(255 - col), B: uint8(col / 2), A: 0xFF}
		if got := Unpremultiply(Premultiply(c)); got != c {
			t.Fatalf("Premultiply roundtrip: got %v want %v", got, c)
		}
	}
}

func TestMixEndpoints(t *testing.T) {
	a, b := color.NRGBA{10, 20, 30, 40}, color.NRGBA{200, 210, 220, 230}
	if got := Mix(a, b, 255); got != a {
		t.Errorf("Mix(a, b, 255) = %v, want %v", got, a)
	}
	if got := Mix(a, b, 0); got != b {
		t.Errorf("Mix(a, b, 0) = %v, want %v", got, b)
	}
	if got := Mix(a, a, 127); got != a {
		t.Errorf("Mix(a, a, 127) = %v, want %v", got, a)
	}
}
