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

// Median applies a separable 3×3 median steps times.
func (ch *Channel) Median(steps int) {
	for i := 0; i < steps; i++ {
		ch.pass3(mid3)
	}
}
