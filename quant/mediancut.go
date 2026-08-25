package quant

import (
	"image"
	"image/color"
	"sort"
)

// MedianCut returns up to n colors chosen by Heckbert's median cut: the color
// box with the widest side is split at its population median until there
// are n boxes, and each box contributes its mean color. n < 1 returns nil;
// an image with fewer distinct colors than n returns them all.
func MedianCut(src *image.NRGBA, n int) color.Palette {
	if n < 1 {
		return nil
	}
	type entry struct {
		key   uint32
		count int
	}
	hist := histogram(src)
	entries := make([]entry, 0, len(hist))
	for k, c := range hist {
		entries = append(entries, entry{k, c})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })

	type box struct {
		lo, hi int // entries[lo:hi]
	}
	channel := func(k uint32, c int) int { return int(k>>(16-8*c)) & 0xff }
	widest := func(b box) (ch, span int) {
		var lo, hi [3]int
		lo = [3]int{255, 255, 255}
		for _, e := range entries[b.lo:b.hi] {
			for c := range 3 {
				v := channel(e.key, c)
				lo[c] = min(lo[c], v)
				hi[c] = max(hi[c], v)
			}
		}
		for c := range 3 {
			if hi[c]-lo[c] > span {
				ch, span = c, hi[c]-lo[c]
			}
		}
		return ch, span
	}

	boxes := []box{{0, len(entries)}}
	for len(boxes) < n {
		// Split the box with the widest side.
		best, bestSpan, bestCh := -1, 0, 0
		for i, b := range boxes {
			if ch, span := widest(b); span > bestSpan {
				best, bestSpan, bestCh = i, span, ch
			}
		}
		if best < 0 {
			break // every box is a single color
		}
		b := boxes[best]
		part := entries[b.lo:b.hi]
		sort.Slice(part, func(i, j int) bool { return channel(part[i].key, bestCh) < channel(part[j].key, bestCh) })
		total := 0
		for _, e := range part {
			total += e.count
		}
		cut, acc := 0, 0
		for cut < len(part)-1 && acc+part[cut].count < total/2 {
			acc += part[cut].count
			cut++
		}
		cut = max(cut, 1)
		boxes[best] = box{b.lo, b.lo + cut}
		boxes = append(boxes, box{b.lo + cut, b.hi})
	}

	p := make(color.Palette, 0, len(boxes))
	for _, b := range boxes {
		var r, g, bl, w int
		for _, e := range entries[b.lo:b.hi] {
			er, eg, eb := unpack(e.key)
			r += int(er) * e.count
			g += int(eg) * e.count
			bl += int(eb) * e.count
			w += e.count
		}
		p = append(p, color.NRGBA{uint8((r + w/2) / w), uint8((g + w/2) / w), uint8((bl + w/2) / w), 255})
	}
	return p
}
