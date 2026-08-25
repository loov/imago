package nrgba

import (
	"image/color"
	"testing"
)

func within(a, b color.NRGBA, tol int) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= tol && int(y)-int(x) <= tol }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

func TestParseString(t *testing.T) {
	for in, want := range map[string]color.NRGBA{
		"#abc": {0xAA, 0xBB, 0xCC, 0xFF}, "#abcd": {0xAA, 0xBB, 0xCC, 0xDD},
		"#1E88E5": {0x1E, 0x88, 0xE5, 0xFF}, "1e88e580": {0x1E, 0x88, 0xE5, 0x80}, "#1E88E580": {0x1E, 0x88, 0xE5, 0x80},
	} {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v", in, got, err)
		}
		if rt, _ := Parse(String(got)); rt != got {
			t.Errorf("round trip %q: %v", in, rt)
		}
	}
	for _, bad := range []string{"", "#12345", "#gg0000"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) succeeded", bad)
		}
	}
	if got := String(Hex(0x1E88E580)); got != "#1e88e580" {
		t.Errorf("String = %q", got)
	}
}

func TestOverContrast(t *testing.T) {
	black, white := RGB8(0, 0, 0), RGB8(255, 255, 255)
	dst, src := color.NRGBA{10, 20, 30, 255}, color.NRGBA{200, 100, 50, 255}
	if got := Over(dst, src); got != src {
		t.Errorf("Over opaque = %v", got)
	}
	if got := Over(dst, color.NRGBA{200, 100, 50, 0}); got != dst {
		t.Errorf("Over transparent = %v", got)
	}
	if got := Over(black, color.NRGBA{255, 255, 255, 128}); !within(got, RGB8(188, 188, 188), 1) {
		t.Errorf("Over 50%% white = %v", got)
	}
	if c := Contrast(white, black); c < 20.95 || c > 21.05 || c != Contrast(black, white) {
		t.Errorf("Contrast = %v / %v", c, Contrast(black, white))
	}
	if TextOn(white) != black || TextOn(black) != white || TextOn(Hex(0x1565C0FF)) != white {
		t.Error("TextOn")
	}
	if got := Ramp(black, white, 3); len(got) != 3 || !within(got[1], RGB8(188, 188, 188), 1) {
		t.Errorf("Ramp = %v", got)
	}
	if Ramp(black, white, 1) != nil {
		t.Error("Ramp n<2")
	}
	a, b := Linear{0, 0.2, 0.4, 1}, Linear{1, 0.8, 0.6, 0.5}
	if a.Lerp(b, -1) != a || a.Lerp(b, 2) != b {
		t.Error("Linear.Lerp endpoints")
	}
	if got := b.MulAlpha(0.5); got != (Linear{0.5, 0.4, 0.3, 0.25}) {
		t.Errorf("MulAlpha = %v", got)
	}
}

func TestTheme(t *testing.T) {
	black, white, red := RGB8(0, 0, 0), RGB8(255, 255, 255), RGB8(255, 0, 0)
	if ToLinear(Lighten(black, 0.5)).Luminance() <= 0 {
		t.Error("Lighten black")
	}
	x := Hex(0x1E88E580)
	if got := Lighten(x, 0); !within(got, x, 1) {
		t.Errorf("Lighten 0 = %v", got)
	}
	if got := Darken(white, 1); !within(got, black, 1) {
		t.Errorf("Darken = %v", got)
	}
	if g := Gray(red); !within(g, RGB8(g.G, g.G, g.G), 2) {
		t.Errorf("Gray = %v", g)
	}
	if s := Shift(red, 1.0/3); s.G <= s.R || s.G <= s.B {
		t.Errorf("Shift = %v", s)
	}
	if got := Saturate(RGB8(128, 128, 128), 1); !within(got, RGB8(128, 128, 128), 1) {
		t.Errorf("Saturate gray = %v", got)
	}
}
