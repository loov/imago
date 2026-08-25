package filter

func mid3(a, b, c byte) byte {
	if a > b {
		if b > c {
			return b
		} else if a < c {
			return a
		}
	} else {
		if a > c {
			return a
		} else if b < c {
			return b
		}
	}
	return c
}

// Median applies a separable 3×3 median steps times. Cost is O(steps·N);
// it is meant for small steps (hot-pixel removal with steps == 1).
func (ch *Channel) Median(steps int) {
	if ch.Width == 0 || ch.Height == 0 {
		return
	}
	for i := 0; i < steps; i++ {
		ch.pass3(mid3)
	}
}
