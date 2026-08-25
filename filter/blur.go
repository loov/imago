package filter

import "math"

// Blur approximates a Gaussian blur of the same variance as steps repeated
// separable 3×3 box blurs, in O(N) regardless of steps.
//
// It runs three separable box blurs of odd width w chosen so that their
// combined variance matches steps 3-tap averages: 3·(w²−1)/12 = 2·steps/3,
// so w = sqrt(8·steps/3 + 1) rounded to the nearest odd integer ≥ 1.
// Edges replicate the border pixel and sums round to nearest. Results differ
// slightly from repeated 3-tap averaging (and Blur(1) is the identity, w = 1).
func (ch *Channel) Blur(steps int) {
	if steps <= 0 || ch.Width == 0 || ch.Height == 0 {
		return
	}
	w := boxWidth(steps)
	if w == 1 {
		return
	}
	r := w / 2
	pad := make([]byte, max(ch.Width, ch.Height)+2*r)
	for range 3 {
		ch.lines(r, pad, func(line []byte, _ int) {
			sum := 0
			for _, v := range pad[:w-1] {
				sum += int(v)
			}
			for i := range line {
				sum += int(pad[i+w-1])
				line[i] = byte((sum + w/2) / w)
				sum -= int(pad[i])
			}
		})
	}
}

// boxWidth returns the odd box width for three passes matching steps 3-tap averages.
func boxWidth(steps int) int {
	return 2*int(math.Round((math.Sqrt(8*float64(steps)/3+1)-1)/2)) + 1
}
