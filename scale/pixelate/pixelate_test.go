package pixelate

import (
	"image"
	"image/color"
	"testing"

	"github.com/loov/imago/pix"
)

func TestResize(t *testing.T) {
	// Left half red, right half blue, with per-pixel noise.
	src := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			n := uint8((x*7 + y*13) % 16)
			c := color.NRGBA{200 + n, n, n, 255}
			if x >= 32 {
				c = color.NRGBA{n, n, 200 + n, 255}
			}
			src.SetNRGBA(x, y, c)
		}
	}
	dst, err := Resize(pix.FromImage(src), 8, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(dst.Palette) != 2 {
		t.Fatalf("palette %v, want 2 colors", dst.Palette)
	}
	if dst.Rect != image.Rect(0, 0, 8, 4) {
		t.Fatalf("bounds %v", dst.Rect)
	}
	for y := range 4 {
		for x := range 8 {
			c := dst.Palette[dst.Pix[x+y*8]].(color.NRGBA)
			if (x < 4) != (c.R > c.B) {
				t.Errorf("pixel (%d,%d) = %v on the wrong side", x, y, c)
			}
		}
	}

	for _, bad := range [][3]int{{0, 4, 2}, {8, 0, 2}, {65, 4, 2}, {8, 33, 2}, {8, 4, 0}} {
		if _, err := Resize(pix.FromImage(src), bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("Resize(%v) returned no error", bad)
		}
	}
}
