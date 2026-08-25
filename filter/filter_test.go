package filter

import "testing"

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
	ch := NewChannel(5, 5)
	ch.Data[2*5+2] = 255
	ch.Blur(1)
	// H pass: 255/3 = 85 on (1..3, 2); V pass: 85/3 = 28 on 3×3.
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			var want byte
			if x >= 1 && x <= 3 && y >= 1 && y <= 3 {
				want = 28
			}
			if got := ch.At(x, y); got != want {
				t.Errorf("(%d,%d) = %d, want %d", x, y, got, want)
			}
		}
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
