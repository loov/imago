package nrgba

import (
	"image/color"
	"testing"

	"github.com/loov/imago/chroma"
)

func TestHSLMatchesChroma(t *testing.T) {
	cases := [][3]float32{
		{0, 1, 0.5}, {1.0 / 3, 1, 0.5}, {2.0 / 3, 1, 0.5}, {1, 1, 0.5}, {-0.25, 1, 0.5},
		{0, 0, 0.5}, {0.1, 0.4, 0.3}, {0.55, 0.7, 0.6}, {0.8, 0.25, 0.8}, {0.45, 1, 0.2},
	}
	for _, c := range cases {
		got := HSL(c[0], c[1], c[2])
		want := FromSRGB(chroma.HSL{H: float64(c[0]), S: float64(c[1]), L: float64(c[2])}.SRGB())
		if !within(got, want, 1) {
			t.Errorf("HSL%v = %v, want %v", c, got, want)
		}
	}
	if got := HSLA(0, 1, 0.5, 0.5); got != (color.NRGBA{R: 255, A: 128}) {
		t.Errorf("HSLA alpha: %v", got)
	}
}

func TestConstructors(t *testing.T) {
	if got := RGBA(1.5, -1, 0.5, 1); got != (color.NRGBA{255, 0, 128, 255}) {
		t.Errorf("RGBA saturate: %v", got)
	}
	r, g, b, a := Floats(color.NRGBA{255, 0, 51, 255})
	if r != 1 || g != 0 || b != 0.2 || a != 1 {
		t.Errorf("Floats: %v %v %v %v", r, g, b, a)
	}
	if Gray8(7) != RGB8(7, 7, 7) || RGB(1, 1, 1) != White || Transparent.A != 0 {
		t.Error("constants")
	}
}
