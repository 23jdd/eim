package eim

import (
	"image/color"
	"testing"
)

func TestColorRGBA(t *testing.T) {
	cases := []Color{
		{},
		{0xff, 0xff, 0xff, 0xff},
		{0x12, 0x34, 0x56, 0x78},
		{0xff, 0x00, 0x80, 0x01},
	}
	for _, c := range cases {
		want := color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
		wr, wg, wb, wa := want.RGBA()
		gr, gg, gb, ga := c.RGBA()
		if gr != wr || gg != wg || gb != wb || ga != wa {
			t.Errorf("Color%v.RGBA(): got (%d,%d,%d,%d); want (%d,%d,%d,%d)", c, gr, gg, gb, ga, wr, wg, wb, wa)
		}
	}
}

func TestHex(t *testing.T) {
	cases := map[string]Color{
		"#fff":      {0xff, 0xff, 0xff, 0xff},
		"ff0000":    {0xff, 0x00, 0x00, 0xff},
		"#00ff0080": {0x00, 0xff, 0x00, 0x80},
		"#zzz":      {},
		"":          {},
	}
	for s, want := range cases {
		if got := Hex(s); got != want {
			t.Errorf("Hex(%q): got %v; want %v", s, got, want)
		}
	}
}

func TestRectIntersect(t *testing.T) {
	r := Rect{X: 0, Y: 0, W: 10, H: 10}
	if got := r.Intersect(Rect{X: 5, Y: 5, W: 10, H: 10}); got != (Rect{5, 5, 5, 5}) {
		t.Errorf("overlap: got %v", got)
	}
	if got := r.Intersect(Rect{X: 20, Y: 20, W: 5, H: 5}); !got.Empty() {
		t.Errorf("no overlap: got %v, want empty", got)
	}
	if got := r.Intersect(Rect{X: -5, Y: -5, W: 20, H: 20}); got != r {
		t.Errorf("contained: got %v; want %v", got, r)
	}
}

func TestRectInset(t *testing.T) {
	r := Rect{X: 0, Y: 0, W: 10, H: 10}.Inset(2)
	if r != (Rect{2, 2, 6, 6}) {
		t.Errorf("Inset(2): got %v", r)
	}
	if got := r.Inset(-1); got != (Rect{1, 1, 8, 8}) {
		t.Errorf("Inset(-1): got %v", got)
	}
}

// TestTextSize 顺带验证默认字体的加载链路（不需要图形上下文）。
func TestTextSize(t *testing.T) {
	ui := &UI{}
	w, h := ui.TextSize("Hello")
	if w <= 0 || h <= 0 {
		t.Fatalf("TextSize: got (%v, %v); want positive", w, h)
	}
	if w2, _ := ui.TextSize("Hello, World"); w2 <= w {
		t.Errorf("longer text should be wider: %v <= %v", w2, w)
	}
}
