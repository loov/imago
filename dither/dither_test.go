package dither

import (
	"image"
	"image/color"
	"testing"
)

func TestBayer(t *testing.T) {
	m := Bayer(4)
	if len(m) != 4 {
		t.Fatalf("size %d", len(m))
	}
	// Every threshold appears once and the set is (0.5/16, 1.5/16, ...).
	seen := map[int]bool{}
	for _, row := range m {
		for _, v := range row {
			k := int(v*16 - 0.5)
			if seen[k] || k < 0 || k > 15 {
				t.Fatalf("bad threshold %v", v)
			}
			seen[k] = true
		}
	}
	if len(Bayer(5)) != 8 || len(Bayer(1)) != 2 {
		t.Error("size rounding")
	}
}

func TestOrdered(t *testing.T) {
	p := color.Palette{color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 255, 255, 255}}
	src := image.NewNRGBA(image.Rect(5, 5, 21, 21))
	for i := range src.Pix {
		src.Pix[i] = 128
		if i%4 == 3 {
			src.Pix[i] = 255
		}
	}
	src.Pix[3] = 0 // first pixel transparent

	// Strength 0 is nearest color: 128 rounds to white everywhere.
	flat := Ordered(src, p, 4, 0)
	if flat.Rect != image.Rect(0, 0, 16, 16) {
		t.Fatalf("bounds %v", flat.Rect)
	}
	for i, v := range flat.Pix[1:] {
		if v != 1 {
			t.Fatalf("pixel %d = %d without dithering", i+1, v)
		}
	}
	// Full strength on mid gray gives a checkerboard-like mix of both.
	mixed := Ordered(src, p, 4, 1)
	var ones int
	for _, v := range mixed.Pix {
		ones += int(v)
	}
	if ones < 64 || ones > 192 {
		t.Errorf("mid gray dithered to %d/256 white", ones)
	}
}
