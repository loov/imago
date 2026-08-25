package filter

// Erode applies a (2·steps+1)² max filter, equivalent to steps repeated
// separable 3×3 max passes with replicated edges, in O(N) regardless of steps
// (van Herk/Gil-Werman).
// It erodes dark features on a light background (grayscale erosion
// of dark lines on light paper); bright features grow.
func (ch *Channel) Erode(steps int) {
	if steps <= 0 || ch.Width == 0 || ch.Height == 0 {
		return
	}
	n := max(ch.Width, ch.Height) + 2*steps
	pad, g, h := make([]byte, n), make([]byte, n), make([]byte, n)
	ch.lines(steps, pad, func(line []byte, m int) {
		k := 2*steps + 1
		for i := 0; i < m; i++ {
			if i%k == 0 {
				g[i] = pad[i]
			} else {
				g[i] = max(g[i-1], pad[i])
			}
		}
		for i := m - 1; i >= 0; i-- {
			if i%k == k-1 || i == m-1 {
				h[i] = pad[i]
			} else {
				h[i] = max(h[i+1], pad[i])
			}
		}
		for i := range line {
			line[i] = max(h[i], g[i+2*steps])
		}
	})
}
