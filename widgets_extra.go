package eim

// 一档扩展控件：不需要 popup / 浮动窗口这类新架构，直接复用现有布局和事件。

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Selectable 绘制一行可选中项（列表行），返回是否被点击。
func (ui *UI) Selectable(label string, selected bool) bool {
	id := ui.ID(label)
	_, th := ui.TextSize(label)
	r := ui.itemRect(0, max(th, ui.style.FrameHeight))

	ev := ui.Hit(Response{Id: id, Area: r})

	switch {
	case selected:
		ui.FillRect(r, ui.style.Accent.WithAlpha(0x3a))
		ui.FillRect(Rect{r.X, r.Y, 2, r.H}, ui.style.Accent)
	case ev.Hovered():
		ui.FillRect(r, ui.style.FrameBGHovered)
	}
	ui.textLeft(r, label, ui.style.FramePadding.X, ui.style.Text)

	ui.advanceItem(r)
	return ev.Clicked()
}

// Switch 绘制一个开关，点击时翻转 *v。
func (ui *UI) Switch(label string, v *bool) bool {
	id := ui.ID(label)
	h := ui.style.FrameHeight - 8
	w := h * 2
	tw, th := ui.TextSize(label)
	r := ui.itemRect(w+ui.style.ItemSpacing.X+tw, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	track := Rect{r.X, r.Y + (r.H-h)/2, w, h}
	bg := ui.style.FrameBG
	switch {
	case *v:
		bg = ui.style.Accent
	case ev.Hovered():
		bg = ui.style.FrameBGHovered
	}
	ui.FillRoundRect(track, h/2, bg)

	knob := h - 6
	kx := track.X + 3
	if *v {
		kx = track.X + w - 3 - knob
	}
	ui.FillCircle(kx+knob/2, track.Y+track.H/2, knob/2, ui.style.CheckMark)

	if label != "" {
		ui.Text(track.X+w+ui.style.ItemSpacing.X, r.Y+(r.H-th)/2, label, ui.style.Text)
	}

	ui.advanceItem(r)
	if ev.Clicked() {
		*v = !*v
		return true
	}
	return false
}

// RadioButton 绘制一个单选项，选中时把 *v 设为 value。
func (ui *UI) RadioButton(label string, v *int, value int) bool {
	id := ui.ID(label)
	d := max(14, ui.style.FrameHeight-10)
	tw, th := ui.TextSize(label)
	r := ui.itemRect(d+ui.style.ItemSpacing.X+tw, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	cx, cy := r.X+d/2, r.Y+r.H/2
	ui.StrokeCircle(cx, cy, d/2-1, 1.5, func() Color {
		if ev.Hovered() {
			return ui.style.FrameBGHovered
		}
		return ui.style.Border
	}())
	if *v == value {
		ui.FillCircle(cx, cy, d/2-4, ui.style.Accent)
	}
	ui.Text(r.X+d+ui.style.ItemSpacing.X, r.Y+(r.H-th)/2, label, ui.style.Text)

	ui.advanceItem(r)
	if ev.Clicked() && *v != value {
		*v = value
		return true
	}
	return false
}

// SeparatorText 绘制一条带标题的分隔线。
func (ui *UI) SeparatorText(label string) {
	ui.NewLine()
	x := ui.layout.cursor.X
	y := ui.layout.cursor.Y
	w := ui.availableWidth()
	tw, th := ui.TextSize(label)

	if w > tw+32 {
		lineY := y + th/2
		left := (w - tw) / 2
		ui.FillRect(Rect{x, lineY, left - 8, 1}, ui.style.Border)
		ui.FillRect(Rect{x + left + tw + 8, lineY, w - left - tw - 8, 1}, ui.style.Border)
		ui.Text(x+left, y, label, ui.style.TextMuted)
	} else {
		ui.Text(x, y, label, ui.style.TextMuted)
	}
	ui.Dummy(Vec2{0, th + 8})
}

// SliderInt 绘制整数滑条。
func (ui *UI) SliderInt(label string, v *int, minV, maxV int) bool {
	if maxV <= minV {
		return false
	}
	id := ui.ID(label)
	r := ui.itemRect(0, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	span := float64(maxV - minV)
	t := clamp01(float32((float64(*v) - float64(minV)) / span))
	changed := false
	if ev.Active() && r.W > 0 {
		nt := clamp01(float32((ui.ctx.Input.Mouse.X - r.X) / r.W))
		if nv := minV + int(math.Round(float64(nt)*span)); nv != *v {
			*v = nv
			changed = true
		}
		t = nt
	}

	ui.drawSliderTrack(r, t, label, fmt.Sprintf("%d", *v))

	ui.advanceItem(r)
	return changed
}

// DragFloat 按住左右拖动调节数值。
func (ui *UI) DragFloat(label string, v *float64, speed float64) bool {
	if speed == 0 {
		speed = 0.01
	}
	id := ui.ID(label)
	r := ui.itemRect(0, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	changed := false
	if ev.Active() {
		if d := float64(ui.ctx.Input.MouseDelta.X) * speed; d != 0 {
			*v += d
			changed = true
		}
	}

	bg := ui.style.FrameBG
	switch {
	case ev.Active():
		bg = ui.style.FrameBGActive
	case ev.Hovered():
		bg = ui.style.FrameBGHovered
	}
	ui.FillRoundRect(r, ui.style.FrameRounding, bg)
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	ui.textLeft(r, label, ui.style.FramePadding.X, ui.style.Text)
	val := fmt.Sprintf("%.3f", *v)
	vw, vh := ui.TextSize(val)
	ui.Text(r.X+r.W-ui.style.FramePadding.X-vw, r.Y+(r.H-vh)/2, val, ui.style.Text)

	ui.advanceItem(r)
	return changed
}

// DragInt 是 DragFloat 的整数版本。
func (ui *UI) DragInt(label string, v *int, speed float64) bool {
	if speed == 0 {
		speed = 1
	}
	id := ui.ID(label)
	r := ui.itemRect(0, ui.style.FrameHeight)

	ev := ui.Hit(Response{Id: id, Area: r})

	changed := false
	if ev.Active() {
		if d := float64(ui.ctx.Input.MouseDelta.X) * speed; d != 0 {
			*v += int(math.Round(d))
			changed = true
		}
	}

	bg := ui.style.FrameBG
	switch {
	case ev.Active():
		bg = ui.style.FrameBGActive
	case ev.Hovered():
		bg = ui.style.FrameBGHovered
	}
	ui.FillRoundRect(r, ui.style.FrameRounding, bg)
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	ui.textLeft(r, label, ui.style.FramePadding.X, ui.style.Text)
	val := fmt.Sprintf("%d", *v)
	vw, vh := ui.TextSize(val)
	ui.Text(r.X+r.W-ui.style.FramePadding.X-vw, r.Y+(r.H-vh)/2, val, ui.style.Text)

	ui.advanceItem(r)
	return changed
}

// TextDisabled 绘制一行灰色文字。
func (ui *UI) TextDisabled(s string) { ui.TextColored(s, ui.style.Disabled) }

// BulletText 绘制一行带圆点的文字。
func (ui *UI) BulletText(s string) {
	w, h := ui.TextSize(s)
	r := ui.itemRect(w+ui.style.IndentSpacing, max(h, ui.textHeight()))
	cy := r.Y + r.H/2
	ui.FillCircle(r.X+4, cy, 2, ui.style.TextMuted)
	ui.Text(r.X+ui.style.IndentSpacing, r.Y, s, ui.style.Text)
	ui.advanceItem(r)
}

// LabelText 在同一行绘制标签和值。
func (ui *UI) LabelText(label, value string) {
	lw, lh := ui.TextSize(label)
	vw, vh := ui.TextSize(value)
	r := ui.itemRect(lw+ui.style.ItemSpacing.X+vw, max(lh, vh))
	ui.Text(r.X, r.Y+(r.H-lh)/2, label, ui.style.TextMuted)
	ui.Text(r.X+lw+ui.style.ItemSpacing.X, r.Y+(r.H-vh)/2, value, ui.style.Text)
	ui.advanceItem(r)
}

// TextWrapped 按当前可用宽度自动换行绘制文本。
func (ui *UI) TextWrapped(s string) {
	w := ui.availableWidth()
	lines := ui.wrapText(s, w)
	lineH := ui.textHeight()
	r := ui.itemRect(w, float32(len(lines))*lineH)
	for i, ln := range lines {
		ui.Text(r.X, r.Y+float32(i)*lineH, ln, ui.style.Text)
	}
	ui.advanceItem(r)
}

// wrapText 按宽度折行，英文优先在空格处断，超长单词（或中文）按字符硬断。
func (ui *UI) wrapText(s string, maxW float32) []string {
	if maxW <= 0 {
		return []string{s}
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, word := range words {
			cand := word
			if line != "" {
				cand = line + " " + word
			}
			if tw, _ := ui.TextSize(cand); tw <= maxW || line == "" {
				line = cand
				// 单个词就超宽：按 rune 硬切
				for {
					tw, _ := ui.TextSize(line)
					if tw <= maxW {
						break
					}
					runes := []rune(line)
					cut := len(runes) - 1
					for cut > 1 {
						if w2, _ := ui.TextSize(string(runes[:cut])); w2 <= maxW {
							break
						}
						cut--
					}
					out = append(out, string(runes[:cut]))
					line = string(runes[cut:])
				}
			} else {
				out = append(out, line)
				line = word
			}
		}
		out = append(out, line)
	}
	return out
}

// Badge 绘制一个胶囊标签，适合配合 SameLine 内联使用。
func (ui *UI) Badge(text string) {
	tw, th := ui.TextSize(text)
	h := th + 6
	r := ui.itemRect(tw+14, h)
	ui.FillRoundRect(r, h/2, ui.style.Accent.WithAlpha(0x33))
	ui.labelIn(r, text, ui.style.Accent)
	ui.advanceItem(r)
}

// Spinner 绘制一个旋转的加载指示器。
func (ui *UI) Spinner() {
	d := ui.style.FrameHeight
	r := ui.itemRect(d, d)
	cx, cy := r.X+d/2, r.Y+d/2
	rad := d/2 - 3

	ui.StrokeCircle(cx, cy, rad, 2, ui.style.Border)
	start := float64(time.Now().UnixMilli()%1000) / 1000 * 2 * math.Pi
	pts := make([]Vec2, 0, 13)
	for i := 0; i <= 12; i++ {
		a := start + float64(i)*math.Pi/12
		pts = append(pts, Vec2{
			X: cx + rad*float32(math.Cos(a)),
			Y: cy + rad*float32(math.Sin(a)),
		})
	}
	ui.StrokePolyline(pts, 2, ui.style.Accent)

	ui.advanceItem(r)
}

// ColorButton 绘制一个色块按钮，返回是否被点击。
func (ui *UI) ColorButton(label string, c Color, size Vec2) bool {
	if size.X <= 0 {
		size.X = ui.style.FrameHeight
	}
	if size.Y <= 0 {
		size.Y = ui.style.FrameHeight
	}
	id := ui.ID(label)
	r := ui.itemRect(size.X, size.Y)

	ev := ui.Hit(Response{Id: id, Area: r})

	ui.FillRoundRect(r, ui.style.FrameRounding, c)
	border := ui.style.Border
	if ev.Hovered() {
		border = ui.style.Text
	}
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, border)

	ui.advanceItem(r)
	return ev.Clicked()
}

// ColorEdit 用 H/S/V 三个滑条编辑颜色，返回是否改变。
func (ui *UI) ColorEdit(label string, c *Color) bool {
	ui.PushID(label)
	defer ui.PopID()

	h, s, v := rgbToHSV(*c)
	changed := ui.SliderFloat("H", &h, 0, 360)
	if ui.SliderFloat("S", &s, 0, 1) {
		changed = true
	}
	if ui.SliderFloat("V", &v, 0, 1) {
		changed = true
	}
	if changed {
		*c = hsvToRGB(h, s, v)
	}
	return changed
}

// RangeSlider 绘制双端滑条，lo/hi 分别是两端手柄。
func (ui *UI) RangeSlider(label string, lo, hi *float64, minV, maxV float64) bool {
	if maxV <= minV {
		return false
	}
	r := ui.itemRect(0, ui.style.FrameHeight)
	if r.W < 20 {
		ui.advanceItem(r)
		return false
	}
	half := r.W / 2
	idLo := ui.ID(label + "##lo")
	idHi := ui.ID(label + "##hi")
	evLo := ui.Hit(Response{Id: idLo, Area: Rect{r.X, r.Y, half, r.H}})
	evHi := ui.Hit(Response{Id: idHi, Area: Rect{r.X + half, r.Y, half, r.H}})

	span := maxV - minV
	toT := func(v float64) float32 { return clamp01(float32((v - minV) / span)) }
	fromX := func(x float32) float64 {
		return minV + float64(clamp01((x-r.X)/r.W))*span
	}

	changed := false
	if evLo.Active() {
		if nv := min(fromX(ui.ctx.Input.Mouse.X), *hi); nv != *lo {
			*lo = nv
			changed = true
		}
	}
	if evHi.Active() {
		if nv := max(fromX(ui.ctx.Input.Mouse.X), *lo); nv != *hi {
			*hi = nv
			changed = true
		}
	}

	tLo, tHi := toT(*lo), toT(*hi)
	xLo, xHi := r.X+r.W*tLo, r.X+r.W*tHi

	ui.FillRoundRect(r, ui.style.FrameRounding, ui.style.FrameBG)
	ui.FillRect(Rect{xLo, r.Y + 2, max(0, xHi-xLo), r.H - 4}, ui.style.Accent.WithAlpha(0x66))
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	for _, x := range []float32{xLo, xHi} {
		ui.FillRoundRect(Rect{x - 3, r.Y + 1, 6, r.H - 2}, 3, ui.style.Accent)
	}

	ui.textLeft(r, label, ui.style.FramePadding.X, ui.style.Text)
	val := fmt.Sprintf("%.2f .. %.2f", *lo, *hi)
	vw, vh := ui.TextSize(val)
	ui.Text(r.X+r.W-ui.style.FramePadding.X-vw, r.Y+(r.H-vh)/2, val, ui.style.Text)

	ui.advanceItem(r)
	return changed
}

// PlotLines 绘制一条迷你折线图（自动缩放到数据范围）。
func (ui *UI) PlotLines(values []float64) {
	r := ui.itemRect(0, ui.style.FrameHeight*2)
	ui.FillRoundRect(r, ui.style.FrameRounding, ui.style.FrameBG)

	if len(values) >= 2 {
		lo, hi := values[0], values[0]
		for _, v := range values[1:] {
			lo = min(lo, v)
			hi = max(hi, v)
		}
		span := hi - lo
		if span == 0 {
			span = 1
		}
		ui.PushClip(r.Inset(2))
		pts := make([]Vec2, len(values))
		for i, v := range values {
			pts[i] = Vec2{
				X: r.X + float32(i)/float32(len(values)-1)*r.W,
				Y: r.Y + r.H - float32((v-lo)/span)*r.H,
			}
		}
		ui.StrokePolyline(pts, 1.5, ui.style.Accent)
		ui.PopClip()
	}

	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	ui.advanceItem(r)
}

// PlotHistogram 绘制一个迷你柱状图。
func (ui *UI) PlotHistogram(values []float64) {
	r := ui.itemRect(0, ui.style.FrameHeight*2)
	ui.FillRoundRect(r, ui.style.FrameRounding, ui.style.FrameBG)

	if len(values) > 0 {
		hi := values[0]
		for _, v := range values[1:] {
			hi = max(hi, v)
		}
		if hi <= 0 {
			hi = 1
		}
		bw := r.W / float32(len(values))
		ui.PushClip(r.Inset(2))
		for i, v := range values {
			h := float32(v/hi) * (r.H - 4)
			ui.FillRect(Rect{r.X + float32(i)*bw + bw*0.15, r.Y + r.H - 2 - h, bw * 0.7, h}, ui.style.Accent.WithAlpha(0xcc))
		}
		ui.PopClip()
	}

	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)
	ui.advanceItem(r)
}

// Knob 绘制一个旋钮，上下拖动调节数值。
func (ui *UI) Knob(label string, v *float64, minV, maxV float64) bool {
	if maxV <= minV {
		return false
	}
	id := ui.ID(label)
	d := ui.style.FrameHeight * 2
	r := ui.itemRect(d, d)

	ev := ui.Hit(Response{Id: id, Area: r})

	changed := false
	if ev.Active() {
		if dy := float64(ui.ctx.Input.MouseDelta.Y); dy != 0 {
			nv := min(max(*v-dy/150*(maxV-minV), minV), maxV)
			if nv != *v {
				*v = nv
				changed = true
			}
		}
	}

	cx, cy := r.X+d/2, r.Y+d/2
	rad := d/2 - 3
	ui.FillCircle(cx, cy, rad, ui.style.FrameBG)
	ui.StrokeCircle(cx, cy, rad, 1, ui.style.Border)

	const startA = -math.Pi * 1.25
	const sweepA = math.Pi * 1.5
	t := clamp01(float32((*v - minV) / (maxV - minV)))

	arc := func(from, to float32, thickness float32, c Color) {
		const n = 24
		pts := make([]Vec2, 0, n+1)
		for i := 0; i <= n; i++ {
			a := from + (to-from)*float32(i)/n
			pts = append(pts, Vec2{
				X: cx + (rad-3)*float32(math.Cos(float64(a))),
				Y: cy + (rad-3)*float32(math.Sin(float64(a))),
			})
		}
		ui.StrokePolyline(pts, thickness, c)
	}
	arc(startA, startA+sweepA, 2, ui.style.Border)
	if t > 0 {
		arc(startA, startA+sweepA*t, 3, ui.style.Accent)
	}
	a := startA + sweepA*t
	ui.Line(cx, cy, cx+(rad-9)*float32(math.Cos(float64(a))), cy+(rad-9)*float32(math.Sin(float64(a))), 2, ui.style.Text)

	ui.advanceItem(r)
	return changed
}

// Tooltip 在鼠标旁显示提示，绘制在所有控件之上（帧末由 App 统一画）。
// 典型用法：if ev.Hovered() { ui.Tooltip("...") }
func (ui *UI) Tooltip(text string) { ui.tip = text }

// drawSliderTrack 是 SliderFloat / SliderInt 共用的滑条外观。
func (ui *UI) drawSliderTrack(r Rect, t float32, label, valText string) {
	ui.FillRoundRect(r, ui.style.FrameRounding, ui.style.FrameBG)
	if t > 0 {
		ui.FillRoundRect(Rect{r.X, r.Y, r.W * t, r.H}, ui.style.FrameRounding, ui.style.Accent.WithAlpha(0x55))
	}
	ui.StrokeRoundRect(r, ui.style.FrameRounding, 1, ui.style.Border)

	const grabW = 4
	gx := min(max(r.X+r.W*t, r.X+grabW/2), r.X+r.W-grabW/2)
	ui.FillRoundRect(Rect{gx - grabW/2, r.Y + 2, grabW, r.H - 4}, 2, ui.style.Accent)

	ui.textLeft(r, label, ui.style.FramePadding.X, ui.style.Text)
	vw, vh := ui.TextSize(valText)
	ui.Text(r.X+r.W-ui.style.FramePadding.X-vw, r.Y+(r.H-vh)/2, valText, ui.style.Text)
}

// WrapText 按 maxW 把文本折行：英文优先在空格断，中文按字符断。
// 供自定义文本控件使用；内置的 TextWrapped 也走它。
func (ui *UI) WrapText(s string, maxW float32) []string { return ui.wrapText(s, maxW) }

// RGBToHSV / HSVToRGB 是颜色空间转换，供自定义取色控件使用。
func RGBToHSV(c Color) (h, s, v float64) { return rgbToHSV(c) }

// HSVToRGB 把 HSV 转回 RGB，h 单位是度。
func HSVToRGB(h, s, v float64) Color { return hsvToRGB(h, s, v) }

// CaretVisible 返回文本光标当前是否处于可见相位（约 1 秒闪烁一次）。
func CaretVisible() bool { return caretVisible() }

// rgbToHSV / hsvToRGB 用于 ColorEdit。
func rgbToHSV(c Color) (h, s, v float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx := max(r, max(g, b))
	mn := min(r, min(g, b))
	v = mx
	if mx > 0 {
		s = (mx - mn) / mx
	}
	if mx != mn {
		switch mx {
		case r:
			h = 60 * math.Mod((g-b)/(mx-mn), 6)
		case g:
			h = 60 * ((b-r)/(mx-mn) + 2)
		default:
			h = 60 * ((r-g)/(mx-mn) + 4)
		}
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}

func hsvToRGB(h, s, v float64) Color {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return Color{
		R: uint8(math.Round((r + m) * 255)),
		G: uint8(math.Round((g + m) * 255)),
		B: uint8(math.Round((b + m) * 255)),
		A: 255,
	}
}
