package quant

import (
	"image"
	"image/color"
	"testing"
)

// fourColors is an image whose pixels take exactly four distinct colors.
func fourColors() (*image.NRGBA, []color.NRGBA) {
	colors := []color.NRGBA{{250, 10, 10, 255}, {10, 250, 10, 255}, {10, 10, 250, 255}, {240, 240, 240, 255}}
	m := image.NewNRGBA(image.Rect(2, 3, 18, 19))
	for y := m.Rect.Min.Y; y < m.Rect.Max.Y; y++ {
		for x := m.Rect.Min.X; x < m.Rect.Max.X; x++ {
			m.SetNRGBA(x, y, colors[(x/4+y/4)%4])
		}
	}
	// A transparent pixel must be ignored.
	m.SetNRGBA(2, 3, color.NRGBA{0, 0, 0, 0})
	return m, colors
}

func contains(p color.Palette, c color.NRGBA) bool {
	for _, e := range p {
		if e.(color.NRGBA) == c {
			return true
		}
	}
	return false
}

func TestQuantizers(t *testing.T) {
	src, want := fourColors()
	for name, fn := range map[string]func(*image.NRGBA, int) color.Palette{
		"MedianCut": MedianCut,
		"Wu":        Wu,
		"KMeans":    func(m *image.NRGBA, n int) color.Palette { return KMeans(m, MedianCut(m, n), 10) },
	} {
		t.Run(name, func(t *testing.T) {
			p := fn(src, 4)
			if len(p) != 4 {
				t.Fatalf("got %d colors, want 4: %v", len(p), p)
			}
			for _, c := range want {
				if !contains(p, c) {
					t.Errorf("palette %v is missing %v", p, c)
				}
			}
			if got := fn(src, 2); len(got) != 2 {
				t.Errorf("n=2: got %d colors", len(got))
			}
			if got := fn(src, 16); len(got) > 4 {
				t.Errorf("n=16 on a 4-color image: got %d colors", len(got))
			}
			if got := fn(src, 0); got != nil {
				t.Errorf("n=0: got %v", got)
			}
			if got := fn(image.NewNRGBA(image.Rect(0, 0, 0, 0)), 4); len(got) != 0 {
				t.Errorf("empty image: got %v", got)
			}
		})
	}
}

func TestKMeansKeepsEmptyClusters(t *testing.T) {
	src, _ := fourColors()
	p := color.Palette{color.NRGBA{250, 10, 10, 255}, color.NRGBA{10, 250, 10, 255}, color.NRGBA{10, 10, 250, 255}, color.NRGBA{240, 240, 240, 255}, color.NRGBA{128, 0, 128, 255}}
	got := KMeans(src, p, 5)
	if len(got) != 5 {
		t.Fatalf("got %d colors, want 5", len(got))
	}
}
