package contentadaptive

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

func TestResize_RejectsInvalidDimensions(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for _, size := range [][2]int{{0, 2}, {2, 0}, {5, 2}, {2, 5}} {
		if _, err := resize(src, size[0], size[1]); err == nil {
			t.Fatalf("Resize(_, %d, %d) returned no error", size[0], size[1])
		}
	}
}

func TestResize_SameSizeCopiesNonZeroBounds(t *testing.T) {
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
			got := dst.NRGBAAt(x, y)
			want := src.NRGBAAt(4+x, 7+y)
			if got != want {
				t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestResize_UniformImageStaysUniform(t *testing.T) {
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
	if dst.Bounds() != image.Rect(0, 0, 3, 2) {
		t.Fatalf("bounds = %v, want (0,0)-(3,2)", dst.Bounds())
	}
	for y := range 2 {
		for x := range 3 {
			got := dst.NRGBAAt(x, y)
			if channelDifference(got.R, want.R) > 1 ||
				channelDifference(got.G, want.G) > 1 ||
				channelDifference(got.B, want.B) > 1 ||
				channelDifference(got.A, want.A) > 1 {
				t.Fatalf("pixel (%d, %d) = %v, want approximately %v", x, y, got, want)
			}
		}
	}
}

func channelDifference(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func grayRow(values ...uint8) *image.NRGBA {
	src := image.NewNRGBA(image.Rect(0, 0, len(values), 1))
	for x, v := range values {
		src.SetNRGBA(x, 0, color.NRGBA{R: v, G: v, B: v, A: 255})
	}
	return src
}

func mustResize(t *testing.T, src image.Image, w, h int) *image.NRGBA {
	t.Helper()
	dst, err := resize(src, w, h)
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// Defect 1: kernel centers and pixel positions used different coordinate origins.
func TestResize_ColumnUniformInputStaysColumnUniform(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for x := range 4 {
		src.SetNRGBA(x, 0, color.NRGBA{A: 255})
		src.SetNRGBA(x, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	}
	dst := mustResize(t, src, 4, 1)
	first := dst.NRGBAAt(0, 0)
	for x := 1; x < 4; x++ {
		if got := dst.NRGBAAt(x, 0); got != first {
			t.Fatalf("column %d = %v, want %v (all columns equal)", x, got, first)
		}
	}
}

// Defect 2: the spatial variance clamp ignored the downscale ratio.
func TestClampCovariance_ScalesWithRatio(t *testing.T) {
	k := kernel{covXX: 100, covYY: 100}
	clampCovariance(&k, 16)
	if k.covXX != maxSpatialVariance*16 || k.covYY != maxSpatialVariance*16 {
		t.Fatalf("clamped to %v, %v; want %v", k.covXX, k.covYY, maxSpatialVariance*16)
	}
}

// Defect 4: color variance was applied as a standard deviation.
func TestResize_GradientNearBoxAverage(t *testing.T) {
	values := make([]uint8, 16)
	for i := range values {
		values[i] = uint8(i * 17)
	}
	dst := mustResize(t, grayRow(values...), 4, 1)
	for x, want := range []uint8{30, 94, 158, 222} {
		if got := dst.NRGBAAt(x, 0).R; channelDifference(got, want) > 8 {
			t.Fatalf("pixel %d = %d, want ~%d", x, got, want)
		}
	}
}

// Defect 5: covariance was accumulated around the stale mean.
func TestMaximize_CovarianceAroundUpdatedMean(t *testing.T) {
	pixels := make([]pixel, 3)
	for i := range pixels {
		pixels[i].alpha = 1
	}
	k := kernel{meanX: 10, meanY: 10, x0: 0, y0: 0, x1: 3, y1: 1, stride: 3, gamma: []float64{1, 0, 1}}
	kernels := []kernel{k}
	maximize(kernels, pixels, 3)
	got := kernels[0]
	if got.meanX != 1.5 || got.covXX != 1 {
		t.Fatalf("meanX, covXX = %v, %v; want 1.5, 1", got.meanX, got.covXX)
	}
}

// Defect 3: fully transparent pixels bled their color into the result.
func TestResize_TransparentPixelsDoNotBleedColor(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{})
	src.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	got := mustResize(t, src, 1, 1).NRGBAAt(0, 0)
	want := color.NRGBA{R: 255, G: 255, B: 255, A: 128}
	if channelDifference(got.R, want.R) > 1 || channelDifference(got.G, want.G) > 1 ||
		channelDifference(got.B, want.B) > 1 || channelDifference(got.A, want.A) > 1 {
		t.Fatalf("got %v, want ~%v", got, want)
	}
}
