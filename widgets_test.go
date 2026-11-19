package eim

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// simFrame 模拟一帧：先 beginFrame（重置 hover），再注入输入，最后渲染。
func simFrame(ctx *Context, in Input, render func()) {
	ctx.Input = in
	ctx.newFrame = true
	ctx.beginFrame()
	render()
}

func clickAt(ctx *Context, render func(), p Vec2) {
	simFrame(ctx, Input{Mouse: p, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}, render)
	simFrame(ctx, Input{Mouse: p, MouseReleased: [3]bool{true}}, render)
}

func TestButtonClick(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	clicks := 0
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		if ui.Button("ok") {
			clicks++
		}
	}

	simFrame(ctx, Input{Mouse: Vec2{10, 10}}, render) // 悬停
	if clicks != 0 {
		t.Fatalf("hover must not click")
	}
	clickAt(ctx, render, Vec2{10, 10})
	if clicks != 1 {
		t.Fatalf("clicked once: got %d", clicks)
	}
	clickAt(ctx, render, Vec2{10, 10})
	if clicks != 2 {
		t.Fatalf("clicked twice: got %d", clicks)
	}
}

func TestButtonClickOutside(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	clicks := 0
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		if ui.Button("ok") {
			clicks++
		}
	}
	clickAt(ctx, render, Vec2{200, 200})
	if clicks != 0 {
		t.Fatalf("click outside must be ignored")
	}
}

func TestCheckboxToggle(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	v := false
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Checkbox("check", &v)
	}

	clickAt(ctx, render, Vec2{10, 10})
	if !v {
		t.Fatalf("checkbox should be on after click")
	}
	clickAt(ctx, render, Vec2{10, 10})
	if v {
		t.Fatalf("checkbox should be off after second click")
	}
}

func TestSliderDrag(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	val := 0.0
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.SliderFloat("s", &val, 0, 1)
	}

	simFrame(ctx, Input{Mouse: Vec2{150, 10}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}, render)
	if val < 0.49 || val > 0.51 {
		t.Fatalf("press at middle: got %v", val)
	}
	simFrame(ctx, Input{Mouse: Vec2{300, 10}, MouseDown: [3]bool{true}}, render)
	if val != 1 {
		t.Fatalf("drag to right edge: got %v", val)
	}
	simFrame(ctx, Input{Mouse: Vec2{150, 10}, MouseReleased: [3]bool{true}}, render)
}

func TestInputText(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	text := ""
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.InputText("t", &text)
	}

	clickAt(ctx, render, Vec2{10, 10})
	if !ui.Focused(ui.ID("t")) {
		t.Fatalf("input should be focused after click")
	}

	simFrame(ctx, Input{Mouse: Vec2{10, 10}, Chars: []rune("hi")}, render)
	if text != "hi" {
		t.Fatalf("typed text: got %q", text)
	}

	simFrame(ctx, Input{Mouse: Vec2{10, 10}, keyBuf: []ebiten.Key{ebiten.KeyBackspace}}, render)
	if text != "h" {
		t.Fatalf("backspace: got %q", text)
	}

	simFrame(ctx, Input{Mouse: Vec2{10, 10}, keyBuf: []ebiten.Key{ebiten.KeyEnter}}, render)
	if ui.Focused(ui.ID("t")) {
		t.Fatalf("enter should drop focus")
	}
	simFrame(ctx, Input{Mouse: Vec2{10, 10}, Chars: []rune("x")}, render)
	if text != "h" {
		t.Fatalf("unfocused input must not receive chars: got %q", text)
	}
}

func TestClickFocusesOnlyOne(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	a, b := "", ""
	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.InputText("a", &a)
		ui.InputText("b", &b)
	}

	// 点击第二个输入框
	clickAt(ctx, render, Vec2{10, 40})
	if ui.Focused(ui.ID("a")) {
		t.Fatalf("a must lose focus")
	}
	if !ui.Focused(ui.ID("b")) {
		t.Fatalf("b should be focused")
	}
	simFrame(ctx, Input{Mouse: Vec2{10, 40}, Chars: []rune("x")}, render)
	if a != "" || b != "x" {
		t.Fatalf("only focused input should receive chars: a=%q b=%q", a, b)
	}
}
