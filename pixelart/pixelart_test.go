package pixelart

import (
	"image"
	"image/color"
	"testing"
)

var (
	black = color.NRGBA{A: 255}
	white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
)

func fill(r image.Rectangle, f func(x, y int) color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetNRGBA(x, y, f(x-r.Min.X, y-r.Min.Y))
		}
	}
	return img
}

func diagonal3() *image.NRGBA {
	return fill(image.Rect(0, 0, 3, 3), func(x, y int) color.NRGBA {
		if x == y {
			return black
		}
		return white
	})
}

func check(t *testing.T, dst *image.NRGBA, want func(x, y int) color.NRGBA) {
	t.Helper()
	for y := range dst.Rect.Dy() {
		for x := range dst.Rect.Dx() {
			if got, w := dst.NRGBAAt(x, y), want(x, y); got != w {
				t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, w)
			}
		}
	}
}

func checkASCII(t *testing.T, dst *image.NRGBA, rows []string) {
	t.Helper()
	check(t, dst, func(x, y int) color.NRGBA {
		if rows[y][x] == '#' {
			return black
		}
		return white
	})
}

func TestRejectsInvalid(t *testing.T) {
	for _, scale := range []func(*image.NRGBA) *image.NRGBA{Scale2x, Scale3x, XBR2x} {
		if dst := scale(nil); dst == nil || dst.Rect != image.Rect(0, 0, 0, 0) {
			t.Fatalf("nil image gave %v", dst)
		}
		if dst := scale(image.NewNRGBA(image.Rect(0, 0, 0, 5))); dst == nil || dst.Rect != image.Rect(0, 0, 0, 0) {
			t.Fatalf("empty image gave %v", dst)
		}
	}
}

func TestUniformAndBounds(t *testing.T) {
	c := color.NRGBA{R: 30, G: 120, B: 220, A: 137}
	src := fill(image.Rect(3, 5, 8, 9), func(x, y int) color.NRGBA { return c })
	for i, scale := range []func(*image.NRGBA) *image.NRGBA{Scale2x, Scale3x, XBR2x} {
		dst := scale(src)
		factor := []int{2, 3, 2}[i]
		if want := image.Rect(0, 0, 5*factor, 4*factor); dst.Rect != want {
			t.Fatalf("bounds = %v, want %v", dst.Rect, want)
		}
		check(t, dst, func(x, y int) color.NRGBA { return c })
	}
}

func TestScale2xCheckerboard(t *testing.T) {
	src := fill(image.Rect(0, 0, 4, 4), func(x, y int) color.NRGBA {
		if (x+y)%2 == 0 {
			return black
		}
		return white
	})
	dst := Scale2x(src)
	// Interior pixels: all four neighbors are equal, so no rule (which needs
	// two adjacent neighbors differing from the other two) fires. Border pixels
	// are excluded since edge clamping makes a clamped neighbor equal the center.
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			if got, want := dst.NRGBAAt(x, y), src.NRGBAAt(x/2, y/2); got != want {
				t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestScale2xDiagonal(t *testing.T) {
	dst := Scale2x(diagonal3())
	// Hand-derived with A=up B=right C=left D=down, clamped at the border.
	// (0,0): A,C clamp to black, B,D white -> E3 = B==D && B!=A && D!=C -> white at (1,1).
	// (0,1): A=(0,0) black, B=(1,1) black, C clamps white, D white -> E1 -> black at (1,2).
	// (1,0): C=(0,0) black, D=(1,1) black, A clamps white, B white -> E2 -> black at (2,1).
	// (1,1): all four neighbors white -> stays black. The rest follow by symmetry.
	checkASCII(t, dst, []string{
		"##....",
		"#.#...",
		".###..",
		"..###.",
		"...#.#",
		"....##",
	})
}

func TestScale3xDiagonal(t *testing.T) {
	dst := Scale3x(diagonal3())
	// Hand-derived with A B C / D E F / G H I, clamped at the border.
	// (0,0): A,B,D clamp black, G clamps to (0,1) white, C,F,H white, I black.
	//   E8: H==F && D!=H && B!=F -> F white at (2,2).
	//   E5: H==F && D!=H && B!=F && E!=C -> F white at (2,1).
	//   E7: H==F && D!=H && B!=F && E!=G -> H white at (1,2).
	// (0,1): B=(0,0) black, F=(1,1) black, D clamps white, H white, A clamps black.
	//   E2: B==F && B!=D && F!=H -> F black at (2,3).
	//   E1: B==F && B!=D && F!=H && E!=A -> B black at (1,3).
	//   E5: needs E!=I, I=(1,2) white -> stays white.
	// (1,1): D,B,F,H all white -> stays black. The rest follow by symmetry.
	checkASCII(t, dst, []string{
		"###......",
		"##.#.....",
		"#..#.....",
		".#####...",
		"...###...",
		"...#####.",
		".....#..#",
		".....#.##",
		"......###",
	})
}

func TestXBR2xDiagonalEdge(t *testing.T) {
	src := fill(image.Rect(2, 3, 8, 9), func(x, y int) color.NRGBA {
		if x+y < 6 {
			return black
		}
		return white
	})
	dst := XBR2x(src)
	differ := 0
	for y := range 12 {
		for x := range 12 {
			if dst.NRGBAAt(x, y) == src.NRGBAAt(2+x/2, 3+y/2) {
				continue
			}
			differ++
			if s := x/2 + y/2; s != 5 && s != 6 {
				t.Fatalf("pixel (%d, %d) changed away from the edge", x, y)
			}
		}
	}
	if differ == 0 {
		t.Fatal("XBR2x produced nearest-neighbor output")
	}
}
