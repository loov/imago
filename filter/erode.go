package filter

func max3(a, b, c byte) byte { return max(a, b, c) }

// Erode applies a separable 3×3 max filter steps times.
// It erodes dark features on a light background (grayscale erosion
// of dark lines on light paper); bright features grow.
func (ch *Channel) Erode(steps int) {
	for i := 0; i < steps; i++ {
		ch.pass3(max3)
	}
}
