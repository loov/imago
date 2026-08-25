package quant

import (
	"image"
	"image/color"

	"github.com/loov/imago/chroma"
)

// KMeans refines palette p against src with Lloyd's algorithm in Oklab for
// at most iterations rounds, stopping early once no pixel changes cluster.
// Colors that end up with no pixels keep their previous value. The result
// has len(p) entries.
func KMeans(src *image.NRGBA, p color.Palette, iterations int) color.Palette {
	if len(p) == 0 {
		return nil
	}
	hist := histogram(src)
	type sample struct {
		c chroma.Oklab
		w float64
	}
	samples := make([]sample, 0, len(hist))
	for k, n := range hist {
		r, g, b := unpack(k)
		samples = append(samples, sample{oklab8(r, g, b), float64(n)})
	}
	centers := make([]chroma.Oklab, len(p))
	for i, c := range p {
		n := color.NRGBAModel.Convert(c).(color.NRGBA)
		centers[i] = oklab8(n.R, n.G, n.B)
	}
	assign := make([]int, len(samples))
	for range iterations {
		changed := false
		for i, s := range samples {
			best, bestD := 0, s.c.Distance(centers[0])
			for j := 1; j < len(centers); j++ {
				if d := s.c.Distance(centers[j]); d < bestD {
					best, bestD = j, d
				}
			}
			if assign[i] != best {
				assign[i], changed = best, true
			}
		}
		if !changed {
			break
		}
		sum := make([]chroma.Oklab, len(centers))
		weight := make([]float64, len(centers))
		for i, s := range samples {
			j := assign[i]
			sum[j].L += s.c.L * s.w
			sum[j].A += s.c.A * s.w
			sum[j].B += s.c.B * s.w
			weight[j] += s.w
		}
		for j := range centers {
			if weight[j] > 0 {
				centers[j] = chroma.Oklab{L: sum[j].L / weight[j], A: sum[j].A / weight[j], B: sum[j].B / weight[j]}
			}
		}
	}
	out := make(color.Palette, len(centers))
	for j, c := range centers {
		r, g, b := chroma.SRGBFromRGB(c.RGB()).Clamp().To8()
		out[j] = color.NRGBA{r, g, b, 255}
	}
	return out
}

func oklab8(r, g, b uint8) chroma.Oklab {
	return chroma.OklabFromRGB(chroma.SRGBFrom8(r, g, b).RGB())
}
