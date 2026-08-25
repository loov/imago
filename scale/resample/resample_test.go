package resample

import (
	"image"
	"image/color"
	"testing"

	"github.com/loov/imago/pix"
)

// resize runs Resize on bytes so the tests keep their byte expectations.
func resize(src image.Image, width, height int, f Filter, linear bool) (*image.NRGBA, error) {
	m := pix.FromImage(src)
	if linear {
		m = m.Linearize()
	}
	dst, err := Resize(m, width, height, f)
	if err != nil {
		return nil, err
	}
	if linear {
		dst = dst.Delinearize()
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

func gray(v uint8) color.NRGBA { return color.NRGBA{R: v, G: v, B: v, A: 255} }

func TestResize(t *testing.T) {
	t.Run("rejects invalid dimensions", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for _, size := range [][2]int{{0, 2}, {2, 0}, {-1, 2}} {
			if _, err := resize(src, size[0], size[1], Lanczos3, false); err == nil {
				t.Fatalf("Resize(_, %d, %d) returned no error", size[0], size[1])
			}
		}
	})

	t.Run("copies non-zero bounds at the same size", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(4, 7, 6, 8))
		src.SetNRGBA(4, 7, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
		src.SetNRGBA(5, 7, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
		dst, err := resize(src, 2, 1, Lanczos3, false)
		if err != nil {
			t.Fatal(err)
		}
		for x := range 2 {
			if got, want := dst.NRGBAAt(x, 0), src.NRGBAAt(4+x, 7); got != want {
				t.Fatalf("pixel %d = %v, want %v", x, got, want)
			}
		}
	})

	t.Run("preserves a uniform image", func(t *testing.T) {
		want := color.NRGBA{R: 30, G: 120, B: 220, A: 137}
		src := fill(image.Rect(3, 5, 11, 13), want)
		for name, f := range map[string]Filter{"lanczos3": Lanczos3, "catmullrom": CatmullRom, "mitchell": MitchellNetravali, "bspline": Mitchell(1, 0)} {
			for _, linear := range []bool{false, true} {
				for _, size := range [][2]int{{3, 2}, {13, 19}} {
					dst, err := resize(src, size[0], size[1], f, linear)
					if err != nil {
						t.Fatal(err)
					}
					for y := range size[1] {
						for x := range size[0] {
							if got := dst.NRGBAAt(x, y); got != want {
								t.Fatalf("%s linear=%v %v: pixel (%d, %d) = %v, want %v", name, linear, size, x, y, got, want)
							}
						}
					}
				}
			}
		}
	})

	t.Run("linear flag changes the average", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
		src.SetNRGBA(0, 0, gray(0))
		src.SetNRGBA(1, 0, gray(255))
		for _, tc := range []struct {
			linear bool
			want   uint8
		}{{false, 128}, {true, 188}} {
			dst, err := resize(src, 1, 1, Lanczos3, tc.linear)
			if err != nil {
				t.Fatal(err)
			}
			if got := dst.NRGBAAt(0, 0); got.R < tc.want-1 || got.R > tc.want+1 || got.A != 255 {
				t.Fatalf("linear=%v: got %v, want gray ~%d", tc.linear, got, tc.want)
			}
		}
	})

	t.Run("b-spline is smoother than catmull-rom on a step edge", func(t *testing.T) {
		src := image.NewNRGBA(image.Rect(0, 0, 4, 1))
		for x := range 4 {
			src.SetNRGBA(x, 0, gray(uint8(255*(x/2))))
		}
		edge := func(f Filter) int {
			dst, err := resize(src, 8, 1, f, false)
			if err != nil {
				t.Fatal(err)
			}
			return int(dst.NRGBAAt(4, 0).R) - int(dst.NRGBAAt(3, 0).R)
		}
		if b, c := edge(Mitchell(1, 0)), edge(CatmullRom); b >= c {
			t.Fatalf("b-spline step %d not smaller than catmull-rom step %d", b, c)
		}
	})
}
