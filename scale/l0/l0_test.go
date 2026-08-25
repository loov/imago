package l0

import (
	"image"
	"image/color"
	"testing"

	"github.com/loov/imago/pix"
)

// resize runs Resize on bytes so the tests keep their byte expectations.
func resize(src image.Image, width, height int, lambda float64) (*image.NRGBA, error) {
	dst, err := Resize(pix.FromImage(src), width, height, lambda)
	if err != nil {
		return nil, err
	}
	return dst.NRGBA(), nil
}

func gray(v uint8) color.NRGBA { return color.NRGBA{R: v, G: v, B: v, A: 255} }

func TestResize(t *testing.T) {
	t.Run("rejects invalid dimensions", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for _, size := range [][2]int{{0, 2}, {2, 0}, {5, 2}, {2, 5}} {
			if _, err := resize(src, size[0], size[1], DefaultLambda); err == nil {
				t.Fatalf("resize(_, %d, %d) returned no error", size[0], size[1])
			}
		}
	})

	t.Run("copies non-zero bounds at the same size", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(4, 7, 6, 9))
		src.SetNRGBA(4, 7, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
		src.SetNRGBA(5, 7, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
		src.SetNRGBA(4, 8, color.NRGBA{R: 90, G: 100, B: 110, A: 120})
		src.SetNRGBA(5, 8, color.NRGBA{R: 130, G: 140, B: 150, A: 160})

		dst, err := resize(src, 2, 2, DefaultLambda)
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

		dst, err := resize(src, 3, 2, DefaultLambda)
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

	t.Run("keeps a step edge sharp", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 8, 2))
		for y := range 2 {
			for x := range 8 {
				if x < 4 {
					src.SetNRGBA(x, y, gray(0))
				} else {
					src.SetNRGBA(x, y, gray(255))
				}
			}
		}
		for _, lambda := range []float64{0, DefaultLambda} {
			dst, err := resize(src, 4, 1, lambda)
			if err != nil {
				t.Fatal(err)
			}
			for x := range 4 {
				got := dst.NRGBAAt(x, 0)
				if x < 2 && got.R > 10 || x >= 2 && got.R < 245 || got.A != 255 {
					t.Fatalf("lambda %v: pixel %d = %v", lambda, x, got)
				}
			}
		}
	})

	t.Run("flattens low-amplitude noise", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		for y := range 8 {
			for x := range 8 {
				v := uint8(124)
				if (x/4+y/4)%2 == 1 {
					v = 132
				}
				src.SetNRGBA(x, y, gray(v))
			}
		}
		dst, err := resize(src, 2, 2, DefaultLambda)
		if err != nil {
			t.Fatal(err)
		}
		for y := range 2 {
			for x := range 2 {
				if got := dst.NRGBAAt(x, y); got.R < 126 || got.R > 130 {
					t.Fatalf("pixel (%d, %d) = %v, want ~128", x, y, got)
				}
			}
		}
		// Without regularization the blocks survive (least squares may slightly over-sharpen).
		dst, err = resize(src, 2, 2, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := dst.NRGBAAt(0, 0); got.R > 124 || got.R < 120 {
			t.Fatalf("lambda 0: pixel (0, 0) = %v, want ~124", got)
		}
	})

	t.Run("transparent pixels do not bleed color", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
		src.SetNRGBA(0, 0, color.NRGBA{})
		src.SetNRGBA(1, 0, gray(255))

		dst, err := resize(src, 1, 1, DefaultLambda)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := dst.NRGBAAt(0, 0), (color.NRGBA{R: 255, G: 255, B: 255, A: 128}); got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}
