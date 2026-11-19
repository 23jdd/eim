package eim

import "testing"

func TestHitClick(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 200, 200})

	resp := Response{Id: 7, Area: Rect{10, 10, 50, 20}}

	// 悬停
	ctx.Input = Input{Mouse: Vec2{20, 20}}
	ev := ui.Hit(resp)
	if !ev.Hovered() || ev.Pressed() || ev.Clicked() || ev.Active() {
		t.Fatalf("hover: hovered=%v pressed=%v clicked=%v", ev.Hovered(), ev.Pressed(), ev.Clicked())
	}

	// 按下
	ctx.Input = Input{Mouse: Vec2{20, 20}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}
	ev = ui.Hit(resp)
	if !ev.Pressed() || !ev.Active() {
		t.Fatalf("press: pressed=%v active=%v", ev.Pressed(), ev.Active())
	}
	if !ui.Focused(7) {
		t.Fatalf("press should focus the item")
	}

	// 在内部松开 -> click
	ctx.hotID = 0
	ctx.Input = Input{Mouse: Vec2{20, 20}, MouseReleased: [3]bool{true}}
	ev = ui.Hit(resp)
	if !ev.Clicked() || !ev.Released() {
		t.Fatalf("release inside: clicked=%v released=%v", ev.Clicked(), ev.Released())
	}
	if ctx.activeID != 0 {
		t.Fatalf("active should be cleared after release")
	}
}

func TestHitDragOutDoesNotClick(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 200, 200})
	resp := Response{Id: 7, Area: Rect{10, 10, 50, 20}}

	ctx.Input = Input{Mouse: Vec2{20, 20}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}
	ui.Hit(resp)

	ctx.hotID = 0
	ctx.Input = Input{Mouse: Vec2{150, 150}, MouseReleased: [3]bool{true}}
	ev := ui.Hit(resp)
	if ev.Clicked() {
		t.Fatalf("dragging out then releasing must not click")
	}
}

func TestHitFirstWins(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 200, 200})
	ctx.Input = Input{Mouse: Vec2{20, 20}}

	a := ui.Hit(Response{Id: 1, Area: Rect{0, 0, 100, 100}})
	b := ui.Hit(Response{Id: 2, Area: Rect{0, 0, 100, 100}})
	if !a.Hovered() {
		t.Fatalf("first item should be hovered")
	}
	if b.Hovered() {
		t.Fatalf("second overlapping item must not steal hover in the same frame")
	}
}

func TestHitRespectsClip(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 200, 200})
	ui.PushClip(Rect{0, 0, 5, 5})
	ctx.Input = Input{Mouse: Vec2{20, 20}}

	if ev := ui.Hit(Response{Id: 1, Area: Rect{0, 0, 100, 100}}); ev.Hovered() {
		t.Fatalf("mouse outside the clip region must not hover")
	}
}

func TestHitActiveBlocksOthers(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 200, 200})
	ctx.Input = Input{Mouse: Vec2{20, 20}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}
	ui.Hit(Response{Id: 1, Area: Rect{0, 0, 100, 100}})

	ctx.hotID = 0
	ctx.Input = Input{Mouse: Vec2{20, 20}, MouseDown: [3]bool{true}}
	if ev := ui.Hit(Response{Id: 2, Area: Rect{0, 0, 100, 100}}); ev.Hovered() {
		t.Fatalf("while item 1 is active, item 2 must not hover")
	}
}

func TestIDScope(t *testing.T) {
	ui := newTestUI()
	base := ui.ID("x")
	ui.PushID("loop")
	scoped := ui.ID("x")
	ui.PopID()

	if base == scoped {
		t.Fatalf("PushID should change the id")
	}
	if got := ui.ID("x"); got != base {
		t.Fatalf("PopID should restore the id: got %v; want %v", got, base)
	}
}
