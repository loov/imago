package nrgba

import (
	"image/color"
	"testing"

	"github.com/loov/imago/chroma"
)

func TestLinearPremultiplyBoundary(t *testing.T) {
	for col := 0; col <= 0xFF; col++ {
		for alpha := 0; alpha <= 0xFF; alpha++ {
			in := color.NRGBA{R: uint8(col), A: uint8(alpha)}
			premul := LinearPremultiply(in)
			if premul.A != uint8(alpha) || premul.R > premul.A {
				t.Fatalf("%v: got %v", in, premul)
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
