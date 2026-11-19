package eim

import (
	"fmt"
	"testing"
	"unicode/utf8"
)

func TestSelectableClick(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	sel := -1
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		for i := 0; i < 3; i++ {
			if ui.Selectable(fmt.Sprintf("row %d", i), sel == i) {
				sel = i
			}
		}
	}

	// 第二行：y = 26+6 = 32 .. 58
	clickAt(ctx, render, Vec2{10, 40})
	if sel != 1 {
		t.Fatalf("selectable: got %d; want 1", sel)
	}
}

func TestSwitchToggle(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	on := false
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Switch("s", &on)
	}

	clickAt(ctx, render, Vec2{10, 10})
	if !on {
		t.Fatalf("switch should be on")
	}
	clickAt(ctx, render, Vec2{10, 10})
	if on {
		t.Fatalf("switch should be off")
	}
}

func TestSliderInt(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	v := 0
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.SliderInt("i", &v, 0, 100)
	}

	simFrame(ctx, Input{Mouse: Vec2{150, 10}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}, render)
	if v != 50 {
		t.Fatalf("slider int: got %d; want 50", v)
	}
	simFrame(ctx, Input{Mouse: Vec2{300, 10}, MouseDown: [3]bool{true}}, render)
	if v != 100 {
		t.Fatalf("slider int right edge: got %d; want 100", v)
	}
}

func TestDragFloat(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	v := 0.0
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.DragFloat("d", &v, 0.01)
	}

	simFrame(ctx, Input{Mouse: Vec2{10, 10}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}, render)
	if v != 0 {
		t.Fatalf("press should not change value: %v", v)
	}
	simFrame(ctx, Input{Mouse: Vec2{15, 10}, MouseDown: [3]bool{true}, MouseDelta: Vec2{5, 0}}, render)
	if v < 0.049 || v > 0.051 {
		t.Fatalf("drag: got %v; want ~0.05", v)
	}
}

func TestRangeSlider(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	lo, hi := 0.2, 0.8
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.RangeSlider("r", &lo, &hi, 0, 1)
	}

	// 左半边按下 -> 拖动 lo，且不能超过 hi
	simFrame(ctx, Input{Mouse: Vec2{30, 10}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}, render)
	simFrame(ctx, Input{Mouse: Vec2{280, 10}, MouseDown: [3]bool{true}}, render)
	if lo != 0.8 {
		t.Fatalf("lo should clamp to hi: got %v", lo)
	}
	if lo > hi {
		t.Fatalf("lo > hi")
	}
}

func TestWrapText(t *testing.T) {
	ui := newTestUI()
	lines := ui.wrapText("the quick brown fox jumps over the lazy dog", 100)
	if len(lines) < 2 {
		t.Fatalf("expected wrapping, got %d lines", len(lines))
	}
	for _, ln := range lines {
		if w, _ := ui.TextSize(ln); w > 100.5 {
			t.Fatalf("line wider than max: %q (%.1f)", ln, w)
		}
	}

	// 中文没有空格，按字符硬断，且不能切坏 UTF-8
	zh := ui.wrapText("这是一段没有空格的很长的中文文本用来测试硬换行", 80)
	if len(zh) < 2 {
		t.Fatalf("cjk should wrap, got %d lines", len(zh))
	}
	for _, ln := range zh {
		if !utf8.ValidString(ln) {
			t.Fatalf("broken utf8: %q", ln)
		}
	}
}

func TestHSV(t *testing.T) {
	cases := []Color{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{0, 0, 255, 255},
		{128, 64, 32, 255},
	}
	for _, c := range cases {
		h, s, v := rgbToHSV(c)
		got := hsvToRGB(h, s, v)
		diff := func(a, b uint8) int {
			d := int(a) - int(b)
			if d < 0 {
				d = -d
			}
			return d
		}
		if diff(got.R, c.R) > 1 || diff(got.G, c.G) > 1 || diff(got.B, c.B) > 1 {
			t.Errorf("hsv round trip: %v -> h=%.1f s=%.2f v=%.2f -> %v", c, h, s, v, got)
		}
	}
}

func TestTooltipFlush(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 300, 300})
	ui.Tooltip("hello")
	if ui.tip != "hello" {
		t.Fatalf("tooltip should be queued")
	}
	ui.flush() // 没有 screen，只验证不 panic 且被消费
	if ui.tip != "" {
		t.Fatalf("tooltip should be consumed")
	}
}

func TestItemHovered(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 300, 300})

	simFrame(ctx, Input{Mouse: Vec2{10, 10}}, func() {
		ui.Button("ok")
		if !ui.ItemHovered() {
			t.Errorf("button under the mouse should be hovered")
		}
	})
	simFrame(ctx, Input{Mouse: Vec2{200, 200}}, func() {
		ui.Button("ok")
		if ui.ItemHovered() {
			t.Errorf("button away from the mouse should not be hovered")
		}
	})
}
