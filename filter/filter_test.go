package filter

import (
	"math/rand"
	"testing"
)

func fill(ch *Channel, v byte) {
	for y := 0; y < ch.Height; y++ {
		for x := 0; x < ch.Width; x++ {
			ch.Data[y*ch.Stride+x] = v
		}
	}
}

func TestDegenerateSizes(t *testing.T) {
	for _, dims := range [][2]int{{1, 1}, {1, 7}, {7, 1}} {
		for _, f := range []func(*Channel, int){(*Channel).Blur, (*Channel).Median, (*Channel).Erode} {
			ch := NewChannel(dims[0], dims[1])
			f(ch, 2)
		}
	}
}

func TestBlurImpulse(t *testing.T) {
	prev := 0
	for _, steps := range []int{1, 3, 10, 40} {
		ch := NewChannel(41, 41)
		ch.Data[20*41+20] = 255
		ch.Blur(steps)
		sum, width := 0, 0
		for y := 0; y < 41; y++ {
			for x := 0; x < 41; x++ {
				v := int(ch.At(x, y))
				sum += v
				if v > 0 {
					width = max(width, x-20)
				}
				if v != int(ch.At(40-x, y)) || v != int(ch.At(x, 40-y)) {
					t.Fatalf("steps %d: asymmetric at (%d,%d)", steps, x, y)
				}
			}
		}
		// Integer rounding per pass compounds for wide boxes; check tight only where values stay large.
		if steps <= 10 && (sum < 255-15 || sum > 255+15) {
			t.Errorf("steps %d: impulse sums to %d", steps, sum)
		}
		if width < prev {
			t.Errorf("steps %d: width %d shrank from %d", steps, width, prev)
		}
		prev = width
	}
	if prev == 0 {
		t.Error("width never grew")
	}

	u := NewChannel(9, 7)
	fill(u, 77)
	u.Blur(5)
	for i, v := range u.Data {
		if v != 77 {
			t.Fatalf("uniform: Data[%d] = %d", i, v)
		}
	}
}

// erodeSlow is the previous implementation: steps repeated 3-tap max passes.
func erodeSlow(ch *Channel, steps int) {
	for i := 0; i < steps; i++ {
		ch.pass3(func(a, b, c byte) byte { return max(a, b, c) })
	}
}

func TestErodeEquivalence(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for iter := 0; iter < 200; iter++ {
		w, h := 1+rng.Intn(30), 1+rng.Intn(30)
		ch := &Channel{Data: make([]byte, (w+3)*h), Width: w, Height: h, Stride: w + 3}
		rng.Read(ch.Data)
		steps := rng.Intn(12)
		want := ch.Clone()
		erodeSlow(want, steps)
		ch.Erode(steps)
		for i := range ch.Data {
			if ch.Data[i] != want.Data[i] {
				t.Fatalf("%dx%d steps %d: Data[%d] = %d, want %d", w, h, steps, i, ch.Data[i], want.Data[i])
			}
		}
	}
}

func TestEmpty(t *testing.T) {
	for _, dims := range [][2]int{{0, 0}, {0, 5}, {5, 0}} {
		ch := NewChannel(dims[0], dims[1])
		ch.Blur(3)
		ch.Median(3)
		ch.Erode(3)
		if ch.Average() != 0 {
			t.Errorf("%v: Average = %v", dims, ch.Average())
		}
	}
	for _, dims := range [][2]int{{-1, 1}, {1, -1}, {1 << 40, 1 << 40}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewChannel%v did not panic", dims)
				}
			}()
			NewChannel(dims[0], dims[1])
		}()
	}
}

func TestMedianHotPixel(t *testing.T) {
	ch := NewChannel(5, 5)
	fill(ch, 100)
	ch.Data[2*5+2] = 255
	ch.Median(1)
	for i, v := range ch.Data {
		if v != 100 {
			t.Errorf("Data[%d] = %d, want 100", i, v)
		}
	}
}

func TestErode(t *testing.T) {
	dark := NewChannel(5, 5)
	fill(dark, 200)
	dark.Data[2*5+2] = 0
	dark.Erode(1)
	for i, v := range dark.Data {
		if v != 200 {
			t.Errorf("dark: Data[%d] = %d, want 200", i, v)
		}
	}

	bright := NewChannel(5, 5)
	bright.Data[2*5+2] = 255
	bright.Erode(1)
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			var want byte
			if x >= 1 && x <= 3 && y >= 1 && y <= 3 {
				want = 255
			}
			if got := bright.At(x, y); got != want {
				t.Errorf("bright: (%d,%d) = %d, want %d", x, y, got, want)
			}
		}
	}
}

func TestAverage(t *testing.T) {
	ch := NewChannel(2, 2)
	copy(ch.Data, []byte{0, 100, 200, 100})
	if got := ch.Average(); got != 100 {
		t.Errorf("Average = %v, want 100", got)
	}
}

func TestStridePadding(t *testing.T) {
	ch := &Channel{Data: make([]byte, 8*4), Width: 5, Height: 4, Stride: 8}
	for i := range ch.Data {
		ch.Data[i] = 7 // padding marker; visible area overwritten below
	}
	fill(ch, 0)
	ch.Data[2*8+2] = 255
	for _, f := range []func(*Channel, int){(*Channel).Blur, (*Channel).Median, (*Channel).Erode} {
		c := ch.Clone()
		f(c, 1)
		for y := 0; y < c.Height; y++ {
			for x := c.Width; x < c.Stride; x++ {
				if c.Data[y*c.Stride+x] != 7 {
					t.Fatalf("padding (%d,%d) modified", x, y)
				}
			}
		}
	}
	if avg := ch.Average(); avg != 255.0/20 {
		t.Errorf("Average = %v, want %v", avg, 255.0/20)
	}
}
