package eim

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// 基础控件。每个控件自己完成：算矩形 -> 命中测试(Hit) -> 绘制 -> 推进布局。
// 所有尺寸都是逻辑单位，DPI 换算由 draw.go 在落笔时统一处理。

// Label 绘制一行文字。
func (ui *UI) Label(s string) {
	w, h := ui.TextSize(s)
	r := ui.itemRect(w, max(h, ui.textHeight()))
	ui.Text(r.X, r.Y, s, ui.style.Text)
	ui.advanceItem(r)
}

// TextColored 绘制一行指定颜色的文字。
func (ui *UI) TextColored(s string, c Color) {
	w, h := ui.TextSize(s)
	r := ui.itemRect(w, max(h, ui.textHeight()))
	ui.Text(r.X, r.Y, s, c)
	ui.advanceItem(r)
}

// Button 绘制一个按钮，返回本帧是否被点击。
func (ui *UI) Button(label string) bool {
	id := ui.ID(label)
	tw, th := ui.TextSize(label)
	h := max(ui.style.FrameHeight, th+2*ui.style.FramePadding.Y)
	w := max(ui.style.MinButtonWidth, tw+2*ui.style.FramePadding.X)
	r := ui.itemRect(w, h)

	ev := ui.Hit(Response{Id: id, Area: r})

	bg := ui.style.FrameBG
	switch {
	case ev.Active():
		bg = ui.style.FrameBGActive
	case ev.Hovered():
		bg = ui.style.FrameBGHovered
	}
	ui.FillRoundRect(r, ui.style.FrameRounding, bg)
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	ui.labelIn(r, label, ui.style.Text)

	ui.advanceItem(r)
	return ev.Clicked()
}

// Checkbox 绘制一个复选框，点击时翻转 *v 并返回 true。
func (ui *UI) Checkbox(label string, v *bool) bool {
	id := ui.ID(label)
	box := max(16, ui.style.FrameHeight-8)
	tw, th := ui.TextSize(label)
	r := ui.itemRect(box+ui.style.ItemSpacing.X+tw, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	bx := Rect{r.X, r.Y + (r.H-box)/2, box, box}
	bg := ui.style.FrameBG
	switch {
	case ev.Active():
		bg = ui.style.FrameBGActive
	case ev.Hovered():
		bg = ui.style.FrameBGHovered
	}
	ui.FillRoundRect(bx, 3, bg)
	ui.StrokeRoundRect(bx, 3, 1, ui.style.Border)
	if *v {
		ui.Line(bx.X+box*0.24, bx.Y+box*0.52, bx.X+box*0.44, bx.Y+box*0.72, 2, ui.style.CheckMark)
		ui.Line(bx.X+box*0.44, bx.Y+box*0.72, bx.X+box*0.78, bx.Y+box*0.28, 2, ui.style.CheckMark)
	}
	ui.Text(bx.X+box+ui.style.ItemSpacing.X, r.Y+(r.H-th)/2, label, ui.style.Text)

	ui.advanceItem(r)
	if ev.Clicked() {
		*v = !*v
		return true
	}
	return false
}

// SliderFloat 绘制一个浮点滑条，拖动时写回 *v，返回本帧值是否改变。
func (ui *UI) SliderFloat(label string, v *float64, minV, maxV float64) bool {
	if maxV <= minV {
		return false
	}
	id := ui.ID(label)
	r := ui.itemRect(0, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	t := clamp01(float32((*v - minV) / (maxV - minV)))
	changed := false
	if ev.Active() && r.W > 0 {
		nt := clamp01(float32((ui.ctx.Input.Mouse.X - r.X) / r.W))
		if nv := minV + float64(nt)*(maxV-minV); nv != *v {
			*v = nv
			changed = true
		}
		t = nt
	}

	ui.drawSliderTrack(r, t, label, fmt.Sprintf("%.2f", *v))

	ui.advanceItem(r)
	return changed
}

// ProgressBar 绘制进度条，fraction 会被夹到 [0,1]。
func (ui *UI) ProgressBar(fraction float32, text string) {
	r := ui.itemRect(0, ui.style.FrameHeight)
	ui.FillRoundRect(r, ui.style.FrameRounding, ui.style.FrameBG)
	if f := clamp01(fraction); f > 0 {
		ui.FillRoundRect(Rect{r.X, r.Y, r.W * f, r.H}, ui.style.FrameRounding, ui.style.Accent)
	}
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	if text != "" {
		ui.labelIn(r, text, ui.style.Text)
	}
	ui.advanceItem(r)
}

// Separator 绘制一条水平分隔线。
func (ui *UI) Separator() {
	ui.NewLine()
	x := ui.layout.cursor.X
	y := ui.layout.cursor.Y + 3
	w := ui.availableWidth()
	ui.FillRect(Rect{x, y, w, 1}, ui.style.Border)
	ui.Dummy(Vec2{0, 7})
}

// InputText 绘制单行输入框。点击获得焦点后可输入，Enter/Esc/Tab 退出编辑。
// 返回本帧文本是否发生变化。
func (ui *UI) InputText(label string, s *string) bool {
	id := ui.ID(label)
	r := ui.itemRect(0, ui.style.FrameHeight)

	ui.Hit(Response{Id: id, Area: r})
	focused := ui.Focused(id)

	changed := false
	if focused && ui.ctx != nil {
		in := &ui.ctx.Input
		for _, ch := range in.Chars {
			if ch < 32 || ch == 127 {
				continue
			}
			*s += string(ch)
			changed = true
		}
		if in.KeyPressed(ebiten.KeyBackspace) && *s != "" {
			rs := []rune(*s)
			*s = string(rs[:len(rs)-1])
			changed = true
		}
		if in.KeyPressed(ebiten.KeyEnter) || in.KeyPressed(ebiten.KeyEscape) || in.KeyPressed(ebiten.KeyTab) {
			ui.ClearFocus()
			focused = false
		}
	}

	bg := ui.style.FrameBG
	if focused {
		bg = ui.style.FrameBGActive
	}
	ui.FillRoundRect(r, ui.style.FrameRounding, bg)
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)

	padX := ui.style.FramePadding.X
	tw, th := ui.TextSize(*s)
	ty := r.Y + (r.H-th)/2
	ui.PushClip(Rect{X: r.X + 1, Y: r.Y, W: r.W - 2, H: r.H})
	ui.Text(r.X+padX, ty, *s, ui.style.Text)
	if focused && caretVisible() {
		cx := r.X + padX + tw
		ui.Line(cx, r.Y+5, cx, r.Y+r.H-5, 1, ui.style.Text)
	}
	ui.PopClip()

	ui.advanceItem(r)
	return changed
}

// Pair 是 Label + 值的只读行，方便做属性列表。
func (ui *UI) Pair(label, value string) {
	_, lh := ui.TextSize(label)
	r := ui.itemRect(0, max(lh, ui.style.FrameHeight))
	vw, vh := ui.TextSize(value)
	ui.Text(r.X, r.Y+(r.H-lh)/2, label, ui.style.TextMuted)
	ui.Text(r.X+r.W-vw, r.Y+(r.H-vh)/2, value, ui.style.Text)
	ui.advanceItem(r)
}

// caretVisible 返回光标是否处于可见相位（约 1 秒闪烁一次）。
func caretVisible() bool {
	return (time.Now().UnixMilli()/530)%2 == 0
}

func clamp01(v float32) float32 { return min(max(v, 0), 1) }
