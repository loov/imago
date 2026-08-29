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
	dst, err := Resize(pix.FromImage(src), 8, 4, Options{Colors: 2})
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
		if _, err := Resize(pix.FromImage(src), bad[0], bad[1], Options{Colors: bad[2]}); err == nil {
			t.Errorf("Resize(%v) returned no error", bad)
		}
	}
}

func TestDither(t *testing.T) {
	// A horizontal gradient with two colors: the middle should get mixed, the
	// ends should stay flat.
	src := image.NewNRGBA(image.Rect(0, 0, 128, 16))
	for y := range 16 {
		for x := range 128 {
			v := uint8(x * 2)
			src.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	dst, err := Resize(pix.FromImage(src), 32, 4, Options{Colors: 2, Dither: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(dst.Palette) != 2 {
		t.Fatalf("palette %v", dst.Palette)
	}
	left, right := dst.Pix[0], dst.Pix[31]
	if left == right {
		t.Fatalf("ends share color %d", left)
	}
	for x := range 4 {
		for y := range 4 {
			if dst.Pix[x+y*32] != left || dst.Pix[31-x+y*32] != right {
				t.Errorf("ends dithered: column %d", x)
			}
		}
	}
	var mixed bool
	for y := range 4 {
		if dst.Pix[15+y*32] != dst.Pix[16+y*32] || dst.Pix[15+y*32] != dst.Pix[15+((y+1)%4)*32] {
			mixed = true
		}
	}
	if !mixed {
		t.Error("middle is not dithered")
	}
}

func TestPalette(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	pal := color.Palette{color.NRGBA{0, 0, 0, 255}, color.NRGBA{240, 240, 240, 255}, color.NRGBA{255, 0, 0, 255}}
	dst, err := Resize(pix.FromImage(src), 4, 4, Options{Palette: pal})
	if err != nil {
		t.Fatal(err)
	}
	if len(dst.Palette) != 3 || dst.Palette[1] != pal[1] {
		t.Fatalf("palette %v", dst.Palette)
	}
	for i, p := range dst.Pix {
		if p != 1 {
			t.Errorf("pixel %d = %d, want 1", i, p)
		}
	}
}
