package eim

import (
	"math"
	"testing"
)

// 逻辑坐标不随 DPI 变；只有落笔时才乘 scale。
func TestUIScale(t *testing.T) {
	ui := newUI(nil, &Context{scale: 2})
	ui.SetRegion(Rect{0, 0, 400, 300})

	if got := ui.Scale(); got != 2 {
		t.Fatalf("Scale: got %v; want 2", got)
	}
	// 样式是逻辑单位
	if got := ui.Style().FrameHeight; got != defaultStyle.FrameHeight {
		t.Errorf("FrameHeight should stay logical: got %v; want %v", got, defaultStyle.FrameHeight)
	}
	// 落笔时换算
	if got, want := ui.px(10), float32(20); got != want {
		t.Errorf("px(10): got %v; want %v", got, want)
	}
	if got, want := ui.pr(Rect{1, 2, 3, 4}), (Rect{2, 4, 6, 8}); got != want {
		t.Errorf("pr: got %v; want %v", got, want)
	}
	// 布局默认行高也不变
	if got := ui.itemRect(0, 0).H; got != defaultStyle.FrameHeight {
		t.Errorf("item height: got %v; want %v", got, defaultStyle.FrameHeight)
	}
}

func TestUIScaleDefault(t *testing.T) {
	ui := newUI(nil, &Context{})
	if got := ui.Scale(); got != 1 {
		t.Fatalf("zero scale should fall back to 1: got %v", got)
	}
}

// TextSize 返回逻辑尺寸，高 DPI 下写控件的人看到的数字应该一样。
func TestTextSizeLogical(t *testing.T) {
	ui1 := newUI(nil, &Context{})
	ui2 := newUI(nil, &Context{scale: 2})

	w1, h1 := ui1.TextSize("Hello")
	w2, h2 := ui2.TextSize("Hello")
	if h1 <= 0 {
		t.Fatalf("font not loaded: h1=%v", h1)
	}
	if math.Abs(float64(w1-w2)) > 1 {
		t.Errorf("text width should be dpi-independent: %v vs %v", w1, w2)
	}
	if math.Abs(float64(h1-h2)) > 1 {
		t.Errorf("text height should be dpi-independent: %v vs %v", h1, h2)
	}
}

func TestSetScale(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetScale(3)
	if got := ui.Scale(); got != 3 {
		t.Fatalf("Scale: got %v; want 3", got)
	}
	// scale 写回 Context，下一帧仍然生效
	ui2 := newUI(nil, ctx)
	if got := ui2.Scale(); got != 3 {
		t.Fatalf("scale should persist in Context: got %v", got)
	}
}
