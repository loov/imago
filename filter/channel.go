// Package filter implements separable 3-tap filters over a byte plane.
//
// Unlike the rest of the module, filters mutate the Channel in place; use
// Clone first to keep the original.
package filter

// Channel is a single byte plane, row-major, with Stride >= Width.
type Channel struct {
	Data   []byte
	Width  int
	Height int
	Stride int
}

// NewChannel allocates a w×h channel with Stride == w.
func NewChannel(w, h int) *Channel {
	return &Channel{Data: make([]byte, w*h), Width: w, Height: h, Stride: w}
}

// At returns the value at (x, y).
func (ch *Channel) At(x, y int) byte { return ch.Data[y*ch.Stride+x] }

// Average returns the mean of all pixels inside Width×Height.
func (ch *Channel) Average() (avg float64) {
	data, w, h, stride := ch.Data, ch.Width, ch.Height, ch.Stride

	for y := 0; y < h; y++ {
		i := y * stride
		e := i + w
		for ; i < e; i++ {
			avg += float64(data[i])
		}
	}

	return avg / float64(w*h)
}

// Clone returns a deep copy.
func (ch *Channel) Clone() *Channel {
	cp := *ch
	cp.Data = make([]byte, len(ch.Data))
	copy(cp.Data, ch.Data)
	return &cp
}

// pass3 applies a 3-tap filter f in place, first horizontally then vertically.
// Edges replicate the border pixel.
func (ch *Channel) pass3(f func(a, b, c byte) byte) {
	data, w, h, stride := ch.Data, ch.Width, ch.Height, ch.Stride

	for y := 0; y < h; y++ {
		i := y * stride
		e := y*stride + w - 1
		p, z := data[i], data[i]
		for ; i < e; i++ {
			n := data[i+1]
			data[i] = f(p, z, n)
			p, z = z, n
		}
		data[i] = f(p, data[i], data[i])
	}

	for x := 0; x < w; x++ {
		i := x
		e := (h-1)*stride + x
		p, z := data[i], data[i]
		for ; i < e; i += stride {
			n := data[i+stride]
			data[i] = f(p, z, n)
			p, z = z, n
		}
		data[i] = f(p, data[i], data[i])
	}
}
