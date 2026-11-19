package eim

import "testing"

func newTestUI() *UI {
	ui := newUI(nil, &Context{})
	ui.SetRegion(Rect{0, 0, 200, 200})
	return ui
}

func TestLayoutVertical(t *testing.T) {
	ui := newTestUI()
	sp := ui.style.ItemSpacing.Y

	r1 := ui.itemRect(50, 20)
	ui.advanceItem(r1)
	r2 := ui.itemRect(50, 20)
	ui.advanceItem(r2)

	if r1 != (Rect{0, 0, 50, 20}) {
		t.Errorf("r1: got %v", r1)
	}
	if r2 != (Rect{0, 20 + sp, 50, 20}) {
		t.Errorf("r2: got %v; want y=%v", r2, 20+sp)
	}
}

func TestLayoutSameLine(t *testing.T) {
	ui := newTestUI()
	sp := ui.style.ItemSpacing

	r1 := ui.itemRect(50, 20)
	ui.advanceItem(r1)

	ui.SameLine()
	r2 := ui.itemRect(30, 16)
	ui.advanceItem(r2)

	if r2.X != 50+sp.X || r2.Y != 0 {
		t.Errorf("r2: got %v; want x=%v y=0", r2, 50+sp.X)
	}

	// 同一行的行高取最大值，下一行从 20 之后开始
	r3 := ui.itemRect(50, 20)
	if r3.Y != 20+sp.Y {
		t.Errorf("r3.y: got %v; want %v", r3.Y, 20+sp.Y)
	}
}

func TestLayoutIndent(t *testing.T) {
	ui := newTestUI()
	ui.Indent(16)
	r := ui.itemRect(50, 20)
	if r.X != 16 {
		t.Errorf("indent x: got %v; want 16", r.X)
	}
	if ui.ContentRegionAvail().W != 200-16 {
		t.Errorf("avail w: got %v; want 184", ui.ContentRegionAvail().W)
	}

	ui.Unindent(16)
	if r := ui.itemRect(50, 20); r.X != 0 {
		t.Errorf("unindent x: got %v; want 0", r.X)
	}
}

func TestLayoutNewLine(t *testing.T) {
	ui := newTestUI()
	r1 := ui.itemRect(50, 20)
	ui.advanceItem(r1)
	ui.SameLine()
	r2 := ui.itemRect(30, 20)
	ui.advanceItem(r2)

	ui.NewLine()
	r3 := ui.itemRect(40, 20)
	if r3.Y != 20+ui.style.ItemSpacing.Y || r3.X != 0 {
		t.Errorf("newline: got %v", r3)
	}
}

func TestChildLayout(t *testing.T) {
	ui := newTestUI()

	ui.BeginChild("child", Vec2{100, 60})
	if got := ui.ClipRect(); got != (Rect{0, 0, 100, 60}) {
		t.Errorf("child clip: got %v", got)
	}
	ui.Label("row")
	// 子区域里的控件从子区域左上角开始
	if got := ui.CursorPos(); got.Y <= 0 {
		t.Errorf("child cursor: got %v", got)
	}
	ui.EndChild()

	// 父布局按子区域高度推进
	if got := ui.CursorPos().Y; got != 60+ui.style.ItemSpacing.Y {
		t.Errorf("parent cursor after child: got %v; want %v", got, 60+ui.style.ItemSpacing.Y)
	}
}

func TestChildScrollClamp(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 200, 200})

	// 鼠标在子区域内、滚轮向上滚很多：滚动值应被夹到 [0, max]
	ctx.Input = Input{Mouse: Vec2{10, 10}, Wheel: Vec2{0, -100}}
	ui.BeginChild("scroll", Vec2{100, 100})
	for i := 0; i < 30; i++ {
		ui.Label("row")
	}
	ui.EndChild()

	maxScroll := ctx.scrollOf(ui.ID("scroll"))
	if maxScroll < 0 {
		t.Fatalf("scroll should not be negative: %v", maxScroll)
	}
	if maxScroll > 30*ui.style.FrameHeight {
		t.Fatalf("scroll should be clamped: %v", maxScroll)
	}
}
