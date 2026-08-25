package ssim

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/loov/imago/pix"
)

func gradient(w, h int) *pix.Image {
	img := pix.New(w, h)
	for y := range h {
		for x := range w {
			v := float64((x+y)*255/(w+h-2)) / 255
			img.Set(x, y, v, v, v, 1)
		}
	}
	return img
}

// shiftedGradient is gradient built through a non-zero-bounds NRGBA.
func shiftedGradient(r image.Rectangle) *pix.Image {
	img := image.NewNRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := uint8(((x - r.Min.X) + (y - r.Min.Y)) * 255 / (r.Dx() + r.Dy() - 2))
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return pix.FromImage(img)
}

func noisy(src *pix.Image, amplitude int) *pix.Image {
	rng := rand.New(rand.NewPCG(1, 2))
	dst := src.Clone()
	for i := range src.Pix {
		if i%4 == 3 {
			continue
		}
		v := int(math.Round(src.Pix[i] * 255))
		dst.Pix[i] = float64(min(max(v+rng.IntN(2*amplitude+1)-amplitude, 0), 255)) / 255
	}
	return dst
}

func checkerboard(w, h, cell int) *pix.Image {
	img := pix.New(w, h)
	for y := range h {
		for x := range w {
			v := float64((x/cell + y/cell) % 2)
			img.Set(x, y, v, v, v, 1)
		}
	}
	return img
}

func inverted(src *pix.Image) *pix.Image {
	dst := src.Clone()
	for i := range src.Pix {
		if i%4 != 3 {
			dst.Pix[i] = 1 - src.Pix[i]
		}
	}
	return dst
}

func TestSSIM(t *testing.T) {
	img := gradient(64, 48)

	t.Run("identical", func(t *testing.T) {
		got, err := SSIM(img, img)
		if err != nil || math.Abs(got-1) > 1e-9 {
			t.Fatalf("SSIM = %v, %v; want 1", got, err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		if _, err := SSIM(nil, img); err == nil {
			t.Fatal("nil image: no error")
		}
		if _, err := SSIM(img, gradient(64, 47)); err == nil {
			t.Fatal("size mismatch: no error")
		}
		small := gradient(10, 10)
		if _, err := SSIM(small, small); err == nil {
			t.Fatal("too small: no error")
		}
	})

	t.Run("inverted", func(t *testing.T) {
		got, err := SSIM(img, inverted(img))
		if err != nil || got >= 0.2 {
			t.Fatalf("SSIM = %v, %v; want < 0.2", got, err)
		}
	})

	t.Run("noise is monotone", func(t *testing.T) {
		prev := 1.0
		for _, amp := range []int{4, 16, 24} {
			got, err := SSIM(img, noisy(img, amp))
			if err != nil || got < 0.5 || got >= prev {
				t.Fatalf("amplitude %d: SSIM = %v, %v; want in [0.5, %v)", amp, got, err, prev)
			}
			prev = got
		}
	})

	t.Run("non-zero bounds", func(t *testing.T) {
		shifted := shiftedGradient(image.Rect(5, 9, 69, 57))
		want, _ := SSIM(img, noisy(img, 16))
		got, err := SSIM(shifted, noisy(shifted, 16))
		if err != nil || got != want {
			t.Fatalf("SSIM = %v, %v; want %v", got, err, want)
		}
	})
}

func TestMSSSIM(t *testing.T) {
	img := gradient(192, 192)

	t.Run("identical", func(t *testing.T) {
		got, err := MSSSIM(img, img)
		if err != nil || math.Abs(got-1) > 1e-9 {
			t.Fatalf("MSSSIM = %v, %v; want 1", got, err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		small := gradient(175, 192)
		if _, err := MSSSIM(small, small); err == nil {
			t.Fatal("too small: no error")
		}
		if _, err := MSSSIM(img, small); err == nil {
			t.Fatal("size mismatch: no error")
		}
	})

	t.Run("checkerboard vs inverse", func(t *testing.T) {
		board := checkerboard(192, 192, 8)
		got, err := MSSSIM(board, inverted(board))
		if err != nil || math.IsNaN(got) || got < 0 || got > 1 {
			t.Fatalf("MSSSIM = %v, %v; want in [0, 1]", got, err)
		}
		got, err = SSIM(board, inverted(board))
		if err != nil || math.IsNaN(got) || got >= 0.2 {
			t.Fatalf("SSIM = %v, %v; want finite < 0.2", got, err)
		}
	})

	t.Run("noise is monotone", func(t *testing.T) {
		prev := 1.0
		for _, amp := range []int{4, 16, 64} {
			got, err := MSSSIM(img, noisy(img, amp))
			if err != nil || got <= 0 || got >= prev {
				t.Fatalf("amplitude %d: MSSSIM = %v, %v; want in (0, %v)", amp, got, err, prev)
			}
			prev = got
		}
	})

	t.Run("non-zero bounds", func(t *testing.T) {
		shifted := shiftedGradient(image.Rect(3, 7, 195, 199))
		want, _ := MSSSIM(img, noisy(img, 16))
		got, err := MSSSIM(shifted, noisy(shifted, 16))
		if err != nil || got != want {
			t.Fatalf("MSSSIM = %v, %v; want %v", got, err, want)
		}
	})
}
