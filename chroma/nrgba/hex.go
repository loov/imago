package nrgba

import (
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"strings"
)

// RGB8 returns an opaque color from 8-bit components.
func RGB8(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 0xFF} }

// Gray8 returns an opaque gray.
func Gray8(v uint8) color.NRGBA { return RGB8(v, v, v) }

// String formats the color as "#rrggbbaa".
func String(c color.NRGBA) string { return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A) }

// Parse accepts "#rgb", "#rgba", "#rrggbb" or "#rrggbbaa", case-insensitive;
// the '#' is optional.
func Parse(s string) (color.NRGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 || len(s) == 4 {
		var b strings.Builder
		for _, r := range s {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		s = b.String()
	}
	if len(s) == 6 {
		s += "ff"
	}
	if len(s) != 8 {
		return color.NRGBA{}, errors.New("nrgba: invalid color length")
	}
	v, err := hex.DecodeString(s)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("nrgba: %w", err)
	}
	return color.NRGBA{R: v[0], G: v[1], B: v[2], A: v[3]}, nil
}
