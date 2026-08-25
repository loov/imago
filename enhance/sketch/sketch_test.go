package sketch

import (
	"image"
	"image/color"
	"testing"
)

func page(w, h int, f func(x, y int) color.NRGBA) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			m.SetNRGBA(x, y, f(x, y))
		}
	}
	return m
}

func gray(v uint8) color.NRGBA { return color.NRGBA{v, v, v, 255} }

func TestReject(t *testing.T) {
	if _, err := Clean(nil, Options{}); err == nil {
		t.Error("nil accepted")
	}
	if _, err := Clean(image.NewNRGBA(image.Rect(0, 0, 0, 5)), Options{}); err == nil {
		t.Error("empty accepted")
	}
}

func TestBoundsAndUniform(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 50, 60))
	for i := range src.Pix {
		src.Pix[i] = 150
		if i%4 == 3 {
			src.Pix[i] = 255
		}
	}
	out, err := Clean(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Rect != image.Rect(0, 0, 40, 40) {
		t.Fatalf("bounds %v", out.Rect)
	}
	for i, v := range out.Pix {
		if v != 255 {
			t.Fatalf("Pix[%d] = %d, want 255", i, v)
		}
	}
	out, _ = Clean(src, Options{Whiteness: 50})
	if v := out.Pix[0]; v < 125 || v > 130 {
		t.Errorf("whiteness 50 gave %d", v)
	}
}

func TestGradientLine(t *testing.T) {
	const w, h = 200, 40
	src := page(w, h, func(x, y int) color.NRGBA {
		if x >= w/2-1 && x <= w/2+1 { // 3px: the median pass erases 1px lines
			return gray(20)
		}
		return gray(uint8(80 + 120*x/(w-1)))
	})
	out, err := Clean(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	lo, hi := 255, 0
	for y := range h {
		for x := range w {
			v := int(out.Pix[out.PixOffset(x, y)])
			if x >= w/2-1 && x <= w/2+1 { // 3px: the median pass erases 1px lines
				if v >= 100 {
					t.Fatalf("line at y=%d is %d", y, v)
				}
				continue
			}
			if (x >= w/2-5 && x <= w/2+5) || x < w/20 || x >= w-w/20 { // skip line halo and erode border
				continue
			}
			lo, hi = min(lo, v), max(hi, v)
		}
	}
	if hi-lo > 8 {
		t.Errorf("background range %d..%d", lo, hi)
	}
}

func TestColor(t *testing.T) {
	src := page(20, 20, func(x, y int) color.NRGBA { return color.NRGBA{200, 150, 150, 255} })
	out, _ := Clean(src, Options{})
	r, g, b := out.Pix[0], out.Pix[1], out.Pix[2]
	if r != g || g != b {
		t.Errorf("not desaturated: %d %d %d", r, g, b)
	}
	out, _ = Clean(src, Options{KeepColor: true, Whiteness: 80})
	r, g, b = out.Pix[0], out.Pix[1], out.Pix[2]
	if r <= g || int(g)-int(b) > 1 || int(b)-int(g) > 1 {
		t.Errorf("tint lost: %d %d %d", r, g, b)
	}
}
