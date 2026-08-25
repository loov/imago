package filter

// Channel is a single byte plane, row-major, with Stride >= Width.
type Channel struct {
	Data   []byte
	Width  int
	Height int
	Stride int
}

// NewChannel allocates a w×h channel with Stride == w.
// It panics on negative dimensions or when w*h overflows.
func NewChannel(w, h int) *Channel {
	if w < 0 || h < 0 || (w > 0 && w*h/w != h) {
		panic("filter: invalid channel size")
	}
	return &Channel{Data: make([]byte, w*h), Width: w, Height: h, Stride: w}
}

// At returns the value at (x, y).
func (ch *Channel) At(x, y int) byte { return ch.Data[y*ch.Stride+x] }

// Average returns the mean of all pixels inside Width×Height, 0 when empty.
func (ch *Channel) Average() (avg float64) {
	data, w, h, stride := ch.Data, ch.Width, ch.Height, ch.Stride
	if w == 0 || h == 0 {
		return 0
	}

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

// lines calls f for every row, then every column, with the line copied into
// pad with r replicated border pixels on each side (m = len(line)+2r). f
// writes its result into line, which aliases ch.Data (stride-aware).
func (ch *Channel) lines(r int, pad []byte, f func(line []byte, m int)) {
	data, w, h, stride := ch.Data, ch.Width, ch.Height, ch.Stride
	line := make([]byte, max(w, h))
	run := func(n int) {
		m := n + 2*r
		copy(pad[r:], line[:n])
		for i := 0; i < r; i++ {
			pad[i], pad[m-1-i] = line[0], line[n-1]
		}
		f(line[:n], m)
	}
	for y := 0; y < h; y++ {
		row := data[y*stride : y*stride+w]
		copy(line, row)
		run(w)
		copy(row, line)
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			line[y] = data[y*stride+x]
		}
		run(h)
		for y := 0; y < h; y++ {
			data[y*stride+x] = line[y]
		}
	}
}
