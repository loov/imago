package box

import (
	"image"
	"image/color"
	"testing"

	"github.com/loov/imago/pix"
)

// resize runs Resize on bytes so the tests keep their byte expectations.
func resize(src image.Image, width, height int) (*image.NRGBA, error) {
	dst, err := Resize(pix.FromImage(src), width, height)
	if err != nil {
		return nil, err
	}
	return dst.NRGBA(), nil
}

func fill(r image.Rectangle, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestResize(t *testing.T) {
	t.Run("rejects invalid dimensions", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for _, size := range [][2]int{{0, 2}, {2, 0}, {5, 2}, {2, 5}} {
			if _, err := resize(src, size[0], size[1]); err == nil {
				t.Fatalf("resize(_, %d, %d) returned no error", size[0], size[1])
			}
		}
	})

	t.Run("copies non-zero bounds at the same size", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(4, 7, 6, 8))
		src.SetNRGBA(4, 7, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
		src.SetNRGBA(5, 7, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
		dst, err := resize(src, 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		for x := range 2 {
			if got, want := dst.NRGBAAt(x, 0), src.NRGBAAt(4+x, 7); got != want {
				t.Fatalf("pixel %d = %v, want %v", x, got, want)
			}
		}
	})

	t.Run("preserves a uniform image at a non-integer scale", func(t *testing.T) {
		want := color.NRGBA{R: 30, G: 120, B: 220, A: 137}
		dst, err := resize(fill(image.Rect(3, 5, 11, 13), want), 3, 2)
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

	t.Run("transparent pixels do not bleed color", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
		src.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		dst, err := resize(src, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := dst.NRGBAAt(0, 0), (color.NRGBA{R: 255, G: 255, B: 255, A: 128}); got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("fractional coverage", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 3, 1))
		src.SetNRGBA(0, 0, color.NRGBA{A: 255})
		src.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		src.SetNRGBA(2, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		dst, err := resize(src, 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		// out0 = (1*0 + 0.5*255)/1.5 = 85, out1 = (0.5*255 + 1*255)/1.5 = 255.
		want := []color.NRGBA{{R: 85, G: 85, B: 85, A: 255}, {R: 255, G: 255, B: 255, A: 255}}
		for x, w := range want {
			if got := dst.NRGBAAt(x, 0); got != w {
				t.Fatalf("pixel %d = %v, want %v", x, got, w)
			}
		}
	})
}
