package chroma

import "math"

// Lab is CIELAB relative to D65, L in 0..100.
type Lab struct{ L, A, B float64 }

// LCh is Lab in polar form, H in turns [0, 1).
type LCh struct{ L, C, H float64 }

const (
	labEps   = 216.0 / 24389 // (6/29)^3
	labKappa = 24389.0 / 27  // (29/3)^3
)

func labF(t float64) float64 {
	if t > labEps {
		return math.Cbrt(t)
	}
	return (labKappa*t + 16) / 116
}

func labFInv(t float64) float64 {
	if t3 := t * t * t; t3 > labEps {
		return t3
	}
	return (116*t - 16) / labKappa
}

// LabFromXYZ converts XYZ to Lab.
func LabFromXYZ(c XYZ) Lab {
	fx, fy, fz := labF(c.X/D65.X), labF(c.Y/D65.Y), labF(c.Z/D65.Z)
	return Lab{L: 116*fy - 16, A: 500 * (fx - fy), B: 200 * (fy - fz)}
}

// XYZ converts Lab to XYZ.
func (c Lab) XYZ() XYZ {
	fy := (c.L + 16) / 116
	fx, fz := fy+c.A/500, fy-c.B/200
	return XYZ{labFInv(fx) * D65.X, labFInv(fy) * D65.Y, labFInv(fz) * D65.Z}
}

// LChFromLab converts to polar form.
func LChFromLab(c Lab) LCh {
	ch, h := polar(c.A, c.B)
	return LCh{c.L, ch, h}
}

// Lab converts to rectangular form.
func (c LCh) Lab() Lab {
	a, b := rect(c.C, c.H)
	return Lab{c.L, a, b}
}

// Distance is the Euclidean distance ΔE76 between two colors.
func (c Lab) Distance(o Lab) float64 {
	return math.Sqrt((c.L-o.L)*(c.L-o.L) + (c.A-o.A)*(c.A-o.A) + (c.B-o.B)*(c.B-o.B))
}
