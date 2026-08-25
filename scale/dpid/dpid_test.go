package dpid

import (
	"image"
	"image/color"
	"math"
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

func TestResize(t *testing.T) {
	t.Run("rejects invalid dimensions", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for _, size := range [][2]int{{0, 2}, {2, 0}, {5, 2}, {2, 5}} {
			if _, err := resize(src, size[0], size[1], 1); err == nil {
				t.Fatalf("resize(_, %d, %d) returned no error", size[0], size[1])
			}
		}
		if _, err := resize(src, 2, 2, -1); err == nil {
			t.Fatal("negative lambda returned no error")
		}
	})

	t.Run("copies non-zero bounds at the same size", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(4, 7, 6, 9))
		src.SetNRGBA(4, 7, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
		src.SetNRGBA(5, 7, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
		src.SetNRGBA(4, 8, color.NRGBA{R: 90, G: 100, B: 110, A: 120})
		src.SetNRGBA(5, 8, color.NRGBA{R: 130, G: 140, B: 150, A: 160})

		dst, err := resize(src, 2, 2, 1)
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

		dst, err := resize(src, 3, 2, 1)
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

	t.Run("lambda zero is a box average", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 8, 2))
		for y := range 2 {
			for x := range 8 {
				v := uint8(x * 30)
				src.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
			}
		}

		dst, err := resize(src, 4, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		for x := range 4 {
			want := uint8(math.Round(float64(2*x*30+(2*x+1)*30) / 2))
			if got := dst.NRGBAAt(x, 0); got != (color.NRGBA{R: want, G: want, B: want, A: 255}) {
				t.Fatalf("pixel %d = %v, want gray %d", x, got, want)
			}
		}
	})

	t.Run("preserves detail", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for y := range 4 {
			for x := range 4 {
				src.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
			}
		}
		src.SetNRGBA(1, 2, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

		box, err := resize(src, 1, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		dst, err := resize(src, 1, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got, avg := dst.NRGBAAt(0, 0), box.NRGBAAt(0, 0); got.R <= avg.R || got.A != 255 {
			t.Fatalf("got %v, want brighter than box average %v", got, avg)
		}
	})

	t.Run("transparent pixels do not bleed color", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
		src.SetNRGBA(0, 0, color.NRGBA{})
		src.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

		dst, err := resize(src, 1, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := dst.NRGBAAt(0, 0), (color.NRGBA{R: 255, G: 255, B: 255, A: 128}); got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("alpha-only detail is preserved", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		src.SetNRGBA(1, 2, color.NRGBA{A: 255})

		dst, err := resize(src, 1, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := dst.NRGBAAt(0, 0); got.A <= uint8(math.Round(255.0/16)) {
			t.Fatalf("got alpha %d, want greater than box average %d", got.A, uint8(math.Round(255.0/16)))
		}
	})
}

func TestSmoothGuide(t *testing.T) {
	impulse := func(w, h, x, y int) [][4]float64 {
		box := make([][4]float64, w*h)
		box[x+y*w] = [4]float64{1, 1, 1, 1}
		return box
	}
	check := func(t *testing.T, box [][4]float64, w, h int, want map[[2]int]float64) {
		t.Helper()
		for y := range h {
			for x := range w {
				if got := smoothGuide(box, x, y, w, h)[0]; math.Abs(got-want[[2]int{x, y}]) > 1e-12 {
					t.Fatalf("guide(%d, %d) = %v, want %v", x, y, got, want[[2]int{x, y}])
				}
			}
		}
	}

	t.Run("interior kernel", func(t *testing.T) {
		// 5x5 so the 3x3 neighborhood of the center is fully interior.
		want := map[[2]int]float64{}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				want[[2]int{2 + dx, 2 + dy}] = float64((2-dx*dx)*(2-dy*dy)) / 16
			}
		}
		check(t, impulse(5, 5, 2, 2), 5, 5, want)
	})

	t.Run("corner renormalizes", func(t *testing.T) {
		check(t, impulse(2, 2, 0, 0), 2, 2, map[[2]int]float64{
			{0, 0}: 4.0 / 9, {1, 0}: 2.0 / 9, {0, 1}: 2.0 / 9, {1, 1}: 1.0 / 9,
		})
	})
}
