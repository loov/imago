package quant

import "image"

// histogram counts distinct colors with non-zero alpha in src, keyed by 0xRRGGBB.
func histogram(src *image.NRGBA) map[uint32]int {
	h := map[uint32]int{}
	if src == nil {
		return h
	}
	w, hgt := src.Rect.Dx(), src.Rect.Dy()
	for y := range hgt {
		row := src.Pix[y*src.Stride : y*src.Stride+4*w]
		for x := range w {
			p := row[4*x : 4*x+4]
			if p[3] == 0 {
				continue
			}
			h[uint32(p[0])<<16|uint32(p[1])<<8|uint32(p[2])]++
		}
	}
	return h
}

func unpack(k uint32) (r, g, b uint8) { return uint8(k >> 16), uint8(k >> 8), uint8(k) }
