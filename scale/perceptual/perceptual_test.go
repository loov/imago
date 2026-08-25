package perceptual

import (
	"image"
	"image/color"
	"testing"

	"github.com/loov/imago/pix"
)

// resize wraps Resize so tests can keep byte-level NRGBA expectations.
func resize(src image.Image, width, height int) (*image.NRGBA, error) {
	dst, err := Resize(pix.FromImage(src), width, height)
	if err != nil {
		return nil, err
	}
	return dst.NRGBA(), nil
}

func TestResize(t *testing.T) {
	t.Run("rejects invalid dimensions", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for _, size := range [][2]int{{0, 2}, {2, 0}, {5, 2}, {2, 5}} {
			if _, err := resize(src, size[0], size[1]); err == nil {
				t.Fatalf("Resize(_, %d, %d) returned no error", size[0], size[1])
			}
		}
	})

	t.Run("copies non-zero bounds at the same size", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(4, 7, 6, 9))
		src.SetNRGBA(4, 7, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
		src.SetNRGBA(5, 7, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
		src.SetNRGBA(4, 8, color.NRGBA{R: 90, G: 100, B: 110, A: 120})
		src.SetNRGBA(5, 8, color.NRGBA{R: 130, G: 140, B: 150, A: 160})

		dst, err := resize(src, 2, 2)
		if err != nil {
			t.Fatal(err)
		}
		for y := range 2 {
			for x := range 2 {
				if got, want := dst.NRGBAAt(x, y), src.NRGBAAt(4+x, 7+y); got != want {
					t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, want)
				}
			}
		}
	})

	t.Run("preserves a uniform image at a non-integer scale", func(t *testing.T) {
		want := color.NRGBA{R: 30, G: 120, B: 220, A: 137}
		src := image.NewNRGBA(image.Rect(3, 5, 11, 13))
		for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
			for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
				src.SetNRGBA(x, y, want)
			}
		}

		dst, err := resize(src, 3, 2)
		if err != nil {
			t.Fatal(err)
		}
		for y := range 2 {
			for x := range 3 {
				if got := dst.NRGBAAt(x, y); got != want {
					t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, want)
				}
			}
		}
	})

	t.Run("restores local contrast", func(t *testing.T) {
		values := [4][4]uint8{
			{0, 0, 255, 255},
			{0, 255, 255, 0},
			{255, 255, 0, 0},
			{255, 0, 0, 255},
		}
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for y := range 4 {
			for x := range 4 {
				v := values[y][x]
				src.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
			}
		}

		dst, err := resize(src, 2, 2)
		if err != nil {
			t.Fatal(err)
		}
		want := [2][2]uint8{{0, 255}, {255, 0}}
		for y := range 2 {
			for x := range 2 {
				if got := dst.NRGBAAt(x, y); got.R != want[y][x] || got.G != want[y][x] || got.B != want[y][x] || got.A != 255 {
					t.Fatalf("pixel (%d, %d) = %v, want gray %d with full alpha", x, y, got, want[y][x])
				}
			}
		}
	})

	t.Run("transparent pixels do not bleed color", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
		src.SetNRGBA(0, 0, color.NRGBA{})
		src.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

		dst, err := resize(src, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := dst.NRGBAAt(0, 0), (color.NRGBA{R: 255, G: 255, B: 255, A: 128}); got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func assertValid(t *testing.T, m *pix.Image) {
	t.Helper()
	for p := 0; p < len(m.Pix); p += 4 {
		a := m.Pix[p+3]
		if a < 0 || a > 1 {
			t.Fatalf("pixel %d: alpha %v out of [0,1]", p/4, a)
		}
		for c := range 3 {
			if v := m.Pix[p+c]; v < 0 || v > a {
				t.Fatalf("pixel %d: channel %d = %v out of [0,%v]", p/4, c, v, a)
			}
		}
	}
}

func TestResizeClampsSinglePixel(t *testing.T) {
	src := pix.New(4, 4)
	src.Set(1, 1, 1, 1, 1, 1)
	dst, err := Resize(src, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	assertValid(t, dst)
}
