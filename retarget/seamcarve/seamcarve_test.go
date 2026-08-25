package seamcarve

import (
	"image"
	"image/color"
	"testing"
)

var black, white = color.NRGBA{A: 255}, color.NRGBA{R: 255, G: 255, B: 255, A: 255}

func fill(r image.Rectangle, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// barImage is a white 6x3 image with a black column 3, transposed if horizontal.
func barImage(horizontal bool) *image.NRGBA {
	img := fill(image.Rect(0, 0, 6, 3), white)
	for y := range 3 {
		img.SetNRGBA(3, y, black)
	}
	if !horizontal {
		return img
	}
	t := image.NewNRGBA(image.Rect(0, 0, 3, 6))
	for y := range 3 {
		for x := range 6 {
			t.SetNRGBA(y, x, img.NRGBAAt(x, y))
		}
	}
	return t
}

func hasBlackColumn(t *testing.T, dst *image.NRGBA, horizontal bool) {
	t.Helper()
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	if horizontal {
		w, h = h, w
	}
	for x := range w {
		all := true
		for y := range h {
			c := dst.NRGBAAt(x, y)
			if horizontal {
				c = dst.NRGBAAt(y, x)
			}
			all = all && c == black
		}
		if all {
			return
		}
	}
	t.Fatalf("no fully black line survived: %v", dst.Pix)
}

func TestResize(t *testing.T) {
	t.Run("rejects invalid dimensions", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for _, size := range [][2]int{{0, 2}, {2, 0}, {5, 2}, {2, 5}} {
			if _, err := Resize(src, size[0], size[1]); err == nil {
				t.Fatalf("Resize(_, %d, %d) returned no error", size[0], size[1])
			}
		}
		if _, err := Resize(nil, 1, 1); err == nil {
			t.Fatal("Resize(nil) returned no error")
		}
	})

	t.Run("copies non-zero bounds at the same size", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(4, 7, 6, 9))
		src.SetNRGBA(4, 7, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
		src.SetNRGBA(5, 7, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
		src.SetNRGBA(4, 8, color.NRGBA{R: 90, G: 100, B: 110, A: 120})
		src.SetNRGBA(5, 8, color.NRGBA{R: 130, G: 140, B: 150, A: 160})

		dst, err := Resize(src, 2, 2)
		if err != nil {
			t.Fatal(err)
		}
		if dst.Rect != image.Rect(0, 0, 2, 2) {
			t.Fatalf("bounds = %v", dst.Rect)
		}
		for y := range 2 {
			for x := range 2 {
				if got, want := dst.NRGBAAt(x, y), src.NRGBAAt(4+x, 7+y); got != want {
					t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, want)
				}
			}
		}
	})

	t.Run("shrinks a uniform image", func(t *testing.T) {
		want := color.NRGBA{R: 30, G: 120, B: 220, A: 137}
		dst, err := Resize(fill(image.Rect(3, 5, 11, 13), want), 3, 2)
		if err != nil {
			t.Fatal(err)
		}
		if dst.Rect != image.Rect(0, 0, 3, 2) {
			t.Fatalf("bounds = %v", dst.Rect)
		}
		for y := range 2 {
			for x := range 3 {
				if got := dst.NRGBAAt(x, y); got != want {
					t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, want)
				}
			}
		}
	})

	t.Run("keeps a vertical bar", func(t *testing.T) {
		dst, err := Resize(barImage(false), 5, 3)
		if err != nil {
			t.Fatal(err)
		}
		hasBlackColumn(t, dst, false)
	})

	t.Run("keeps a horizontal bar", func(t *testing.T) {
		dst, err := Resize(barImage(true), 3, 5)
		if err != nil {
			t.Fatal(err)
		}
		hasBlackColumn(t, dst, true)
	})

	t.Run("removes both a column and a row", func(t *testing.T) {
		dst, err := Resize(barImage(false), 5, 2)
		if err != nil {
			t.Fatal(err)
		}
		if dst.Rect != image.Rect(0, 0, 5, 2) {
			t.Fatalf("bounds = %v", dst.Rect)
		}
		hasBlackColumn(t, dst, false)
	})
}
