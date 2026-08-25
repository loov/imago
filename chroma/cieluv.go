package chroma

import "math"

// Luv is CIELUV relative to D65, L in 0..100.
type Luv struct{ L, U, V float64 }

// uvPrime returns u', v' chromaticity; (0,0) for black.
func uvPrime(c XYZ) (u, v float64) {
	d := c.X + 15*c.Y + 3*c.Z
	if d == 0 {
		return 0, 0
	}
	return 4 * c.X / d, 9 * c.Y / d
}

// LuvFromXYZ converts XYZ to Luv.
func LuvFromXYZ(c XYZ) Luv {
	y := c.Y / d65Y
	var l float64
	if y > labEps {
		l = 116*math.Cbrt(y) - 16
	} else {
		l = labKappa * y
	}
	u, v := uvPrime(c)
	un, vn := uvPrime(XYZ{d65X, d65Y, d65Z})
	return Luv{L: l, U: 13 * l * (u - un), V: 13 * l * (v - vn)}
}

// XYZ converts Luv to XYZ.
func (c Luv) XYZ() XYZ {
	if c.L == 0 {
		return XYZ{}
	}
	var y float64
	if c.L > labKappa*labEps {
		t := (c.L + 16) / 116
		y = t * t * t
	} else {
		y = c.L / labKappa
	}
	y *= d65Y
	un, vn := uvPrime(XYZ{d65X, d65Y, d65Z})
	u, v := c.U/(13*c.L)+un, c.V/(13*c.L)+vn
	x := y * 9 * u / (4 * v)
	z := y * (12 - 3*u - 20*v) / (4 * v)
	return XYZ{x, y, z}
}
