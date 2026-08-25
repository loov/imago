package l0

import (
	"image"
	"image/color"
	"math"
	"math/rand"
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

func TestResizeClampsStepEdge(t *testing.T) {
	src := pix.New(8, 2)
	for y := range 2 {
		for x := range 8 {
			v := float64(x / 4)
			src.Set(x, y, v, v, v, 1)
		}
	}
	for _, lambda := range []float64{0, DefaultLambda} {
		dst, err := Resize(src, 4, 1, lambda)
		if err != nil {
			t.Fatal(err)
		}
		assertValid(t, dst)
	}
}

func TestResizeRejectsNonFiniteLambda(t *testing.T) {
	src := pix.New(4, 4)
	for _, lambda := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := Resize(src, 2, 2, lambda); err == nil {
			t.Fatalf("lambda %v accepted", lambda)
		}
	}
}

func TestResizeConverges(t *testing.T) {
	src := pix.New(8, 8)
	for p := 0; p < len(src.Pix); p += 4 {
		src.Pix[p], src.Pix[p+1], src.Pix[p+2], src.Pix[p+3] = 0.5, 0.5, 0.5, 1
	}
	if _, err := Resize(src, 2, 2, DefaultLambda); err != nil {
		t.Fatal(err)
	}
	if iterations >= 8 {
		t.Fatalf("uniform image took %d outer iterations", iterations)
	}
}

func BenchmarkResize_1080p(b *testing.B) {
	src := pix.New(1920, 1080)
	for y := range 1080 {
		for x := range 1920 {
			src.Set(x, y, float64(x)/1920, float64(y)/1080, 0.5, 1)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Resize(src, 480, 270, DefaultLambda); err != nil {
			b.Fatal(err)
		}
	}
}

// TestUpsamplerMatchesReference checks the separable tables against a
// straightforward per-pixel bilinear evaluation of U and Uᵀ.
func TestUpsamplerMatchesReference(t *testing.T) {
	const iw, ih, w, h = 13, 7, 5, 3
	rng := rand.New(rand.NewSource(1))
	d := make([]float64, w*h)
	for i := range d {
		d[i] = rng.Float64()
	}
	input := make([]float64, iw*ih)
	for i := range input {
		input[i] = rng.Float64()
	}
	// Reference: Uᵀ(Ud − input) and column sums, computed per input pixel.
	wantGrad, wantNorm := make([]float64, w*h), make([]float64, w*h)
	for y := range ih {
		sy := min(max((float64(y)+0.5)*h/ih-0.5, 0), float64(h-1))
		y0 := int(sy)
		y1 := min(y0+1, h-1)
		fy := sy - float64(y0)
		for x := range iw {
			sx := min(max((float64(x)+0.5)*w/iw-0.5, 0), float64(w-1))
			x0 := int(sx)
			x1 := min(x0+1, w-1)
			fx := sx - float64(x0)
			idx := [4]int{x0 + y0*w, x1 + y0*w, x0 + y1*w, x1 + y1*w}
			wt := [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}
			r := -input[x+y*iw]
			for k := range 4 {
				r += wt[k] * d[idx[k]]
			}
			for k := range 4 {
				wantGrad[idx[k]] += wt[k] * r
				wantNorm[idx[k]] += wt[k]
			}
		}
	}
	up := newUpsampler(iw, ih, w, h)
	grad := make([]float64, w*h)
	up.residualT(d, input, grad, w)
	for i := range grad {
		if math.Abs(grad[i]-wantGrad[i]) > 1e-12 || math.Abs(up.norm[i]-wantNorm[i]) > 1e-12 {
			t.Fatalf("pixel %d: grad %v want %v, norm %v want %v", i, grad[i], wantGrad[i], up.norm[i], wantNorm[i])
		}
	}
}
