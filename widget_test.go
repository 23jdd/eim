package eim

import "testing"

// 自定义控件只用导出 API：ItemRect + Hit + 绘制 + 返回 Response。
func TestCustomWidget(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)

	clicks := 0
	w := WidgetFunc(func(ui *UI) Response {
		id := ui.ID("custom")
		r := ui.ItemRect(100, 20)
		if ui.Hit(Response{Id: id, Area: r}).Clicked() {
			clicks++
		}
		return Response{Id: id, Area: r}
	})

	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Add(w)
	}

	clickAt(ctx, render, Vec2{10, 10})
	if clicks != 1 {
		t.Fatalf("custom widget click: got %d", clicks)
	}
	// Add 按返回的 Area 推进了布局
	if got, want := ui.CursorPos().Y, 20+ui.style.ItemSpacing.Y; got != want {
		t.Fatalf("layout advance: got %v; want %v", got, want)
	}
}

// 自定义控件也能参与 SameLine 和 hover。
func TestCustomWidgetSameLine(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)

	var rightR Rect
	hoveredLeft, hoveredRight := false, false

	left := WidgetFunc(func(ui *UI) Response {
		r := ui.ItemRect(60, 24)
		if ui.Hit(Response{Id: ui.ID("left"), Area: r}).Hovered() {
			hoveredLeft = true
		}
		return Response{Id: ui.ID("left"), Area: r}
	})
	right := WidgetFunc(func(ui *UI) Response {
		r := ui.ItemRect(60, 24)
		rightR = r
		if ui.Hit(Response{Id: ui.ID("right"), Area: r}).Hovered() {
			hoveredRight = true
		}
		return Response{Id: ui.ID("right"), Area: r}
	})

	simFrame(ctx, Input{Mouse: Vec2{30, 10}}, func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Add(left)
		ui.SameLine()
		ui.Add(right)
	})

	if rightR.X != 60+ui.style.ItemSpacing.X || rightR.Y != 0 {
		t.Fatalf("SameLine placement: right=%v", rightR)
	}
	if !hoveredLeft || hoveredRight {
		t.Fatalf("hover: left=%v right=%v", hoveredLeft, hoveredRight)
	}
	if got := ui.CursorPos().X; got != 0 {
		t.Fatalf("cursor should reset to line start: got %v", got)
	}
}

// ItemRect 的默认值：宽 <=0 占满剩余，高 <=0 用 FrameHeight。
func TestItemRectDefaults(t *testing.T) {
	ui := newTestUI()
	r := ui.ItemRect(0, 0)
	if r.W != 200 || r.H != ui.style.FrameHeight {
		t.Fatalf("ItemRect(0,0): got %v", r)
	}
}

// 只用导出 API 从零实现一个 Button，行为应与内置 Button 一致。
func TestCustomButtonParity(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)

	clicks := 0
	btn := WidgetFunc(func(ui *UI) Response {
		id := ui.ID("ok")
		tw, th := ui.TextSize("ok")
		r := ui.ItemRect(tw+2*ui.Style().FramePadding.X, max(th+2*ui.Style().FramePadding.Y, ui.Style().FrameHeight))

		ev := ui.Hit(Response{Id: id, Area: r})

		bg := ui.Style().FrameBG
		switch {
		case ev.Active():
			bg = ui.Style().FrameBGActive
		case ev.Hovered():
			bg = ui.Style().FrameBGHovered
		}
		ui.FillRoundRect(r, ui.Style().FrameRounding, bg)
		ui.StrokeRoundRect(r, ui.Style().FrameRounding, 1, ui.Style().Border)
		ui.TextCentered(r, "ok", ui.Style().Text)

		if ev.Clicked() {
			clicks++
		}
		return Response{Id: id, Area: r}
	})

	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Add(btn)
	}

	clickAt(ctx, render, Vec2{10, 10})
	if clicks != 1 {
		t.Fatalf("custom button click: got %d", clicks)
	}
	// 点击在控件外不应触发
	clickAt(ctx, render, Vec2{250, 250})
	if clicks != 1 {
		t.Fatalf("click outside must not fire: got %d", clicks)
	}
}

// 用 ui.Input() 实现文本输入，验证字符流和焦点。
func TestCustomInputTextParity(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)

	text := ""
	field := WidgetFunc(func(ui *UI) Response {
		id := ui.ID("field")
		r := ui.ItemRect(0, ui.Style().FrameHeight)

		ui.Hit(Response{Id: id, Area: r})
		if in := ui.Input(); in != nil && ui.Focused(id) {
			for _, ch := range in.Chars {
				text += string(ch)
			}
		}
		return Response{Id: id, Area: r}
	})

	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Add(field)
	}

	clickAt(ctx, render, Vec2{10, 10})
	simFrame(ctx, Input{Mouse: Vec2{10, 10}, Chars: []rune("ab")}, render)
	if text != "ab" {
		t.Fatalf("custom input: got %q", text)
	}
}

// 用 ui.Input().MouseDelta 实现拖拽。
func TestCustomDragParity(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)

	val := 0.0
	drag := WidgetFunc(func(ui *UI) Response {
		id := ui.ID("drag")
		r := ui.ItemRect(0, ui.Style().FrameHeight)
		ev := ui.Hit(Response{Id: id, Area: r})
		if ev.Active() {
			if in := ui.Input(); in != nil {
				val += float64(in.MouseDelta.X) * 0.1
			}
		}
		return Response{Id: id, Area: r}
	})

	render := func() {
		ui.SetRegion(Rect{0, 0, 300, 300})
		ui.Add(drag)
	}

	simFrame(ctx, Input{Mouse: Vec2{10, 10}, MouseDown: [3]bool{true}, MousePressed: [3]bool{true}}, render)
	simFrame(ctx, Input{Mouse: Vec2{20, 10}, MouseDown: [3]bool{true}, MouseDelta: Vec2{10, 0}}, render)
	if val < 0.99 || val > 1.01 {
		t.Fatalf("custom drag: got %v; want ~1.0", val)
	}
}

func TestInputNilContext(t *testing.T) {
	ui := newUI(nil, nil)
	if ui.Input() != nil {
		t.Fatalf("Input should be nil without a Context")
	}
}
