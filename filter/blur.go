package filter

func avg3(a, b, c byte) byte {
	return byte((int(a) + int(b) + int(c)) / 3)
}

// Blur applies a separable 3×3 box blur steps times.
// Repeated box blurs approximate a Gaussian blur.
func (ch *Channel) Blur(steps int) {
	for i := 0; i < steps; i++ {
		ch.pass3(avg3)
	}
}
