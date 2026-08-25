package chroma

// RGB is linear light in sRGB primaries. It is the hub for SRGB and Oklab.
type RGB struct{ R, G, B float64 }

// XYZ is CIE 1931 XYZ relative to D65. It is the hub for Lab and Luv.
type XYZ struct{ X, Y, Z float64 }

// D65 reference white in XYZ, used by Lab and Luv.
const d65X, d65Y, d65Z = 0.95047, 1.0, 1.08883

// mat is a row-major 3x3 matrix.
type mat [3][3]float64

func (m mat) mul(a, b, c float64) (float64, float64, float64) {
	return m[0][0]*a + m[0][1]*b + m[0][2]*c,
		m[1][0]*a + m[1][1]*b + m[1][2]*c,
		m[2][0]*a + m[2][1]*b + m[2][2]*c
}

// inv returns the inverse; used so round-trips are exact to float precision
// rather than limited by published truncated inverses.
func (m mat) inv() mat {
	c := mat{
		{m[1][1]*m[2][2] - m[1][2]*m[2][1], m[0][2]*m[2][1] - m[0][1]*m[2][2], m[0][1]*m[1][2] - m[0][2]*m[1][1]},
		{m[1][2]*m[2][0] - m[1][0]*m[2][2], m[0][0]*m[2][2] - m[0][2]*m[2][0], m[0][2]*m[1][0] - m[0][0]*m[1][2]},
		{m[1][0]*m[2][1] - m[1][1]*m[2][0], m[0][1]*m[2][0] - m[0][0]*m[2][1], m[0][0]*m[1][1] - m[0][1]*m[1][0]},
	}
	det := m[0][0]*c[0][0] + m[0][1]*c[1][0] + m[0][2]*c[2][0]
	for i := range c {
		for j := range c[i] {
			c[i][j] /= det
		}
	}
	return c
}

var (
	rgbToXYZ = mat{
		{0.4124564, 0.3575761, 0.1804375},
		{0.2126729, 0.7151522, 0.0721750},
		{0.0193339, 0.1191920, 0.9503041},
	}
	xyzToRGB = rgbToXYZ.inv()
)

// XYZ converts linear RGB to XYZ.
func (c RGB) XYZ() XYZ {
	x, y, z := rgbToXYZ.mul(c.R, c.G, c.B)
	return XYZ{x, y, z}
}

// RGBFromXYZ converts XYZ to linear RGB.
func RGBFromXYZ(c XYZ) RGB {
	r, g, b := xyzToRGB.mul(c.X, c.Y, c.Z)
	return RGB{r, g, b}
}

// Clamp limits each component to 0..1.
func (c RGB) Clamp() RGB {
	return RGB{min(max(c.R, 0), 1), min(max(c.G, 0), 1), min(max(c.B, 0), 1)}
}
