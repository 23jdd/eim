package eim

// 布局：光标式（immediate mode）线性布局，规则和 Dear ImGui 一致。
//
//	默认从上往下堆，每个 item 画完光标移到下一行；
//	SameLine 让下一个 item 回到当前行右侧；
//	Indent 改左缩进；
//	BeginChild/EndChild 开一个可独立裁剪、独立滚动的子区域。

// layout 是当前容器内的光标状态。
type layout struct {
	region Rect    // 容器内容区（屏幕坐标）
	cursor Vec2    // 下一个 item 的左上角
	lineY  float32 // 当前行的 Y
	lineH  float32 // 当前行已占用的最大高度
	lastX  float32 // 当前行最后一个 item 的右边缘
	indent float32
}

// childState 记录一个打开中的子区域。
type childState struct {
	id     int64
	rect   Rect
	scroll float32
}

const childScrollSpeed = 40

// SetRegion 把布局区域设为 r，并重置光标。用于把 UI 嵌进某块区域。
func (ui *UI) SetRegion(r Rect) {
	ui.layout = layout{
		region: r,
		cursor: Vec2{r.X, r.Y},
		lineY:  r.Y,
		lastX:  r.X,
	}
}

// CursorPos 返回下一个 item 的左上角。
func (ui *UI) CursorPos() Vec2 { return ui.layout.cursor }

// SetCursorPos 移动光标。
func (ui *UI) SetCursorPos(p Vec2) {
	ui.layout.cursor = p
	ui.layout.lineY = p.Y
	ui.layout.lineH = 0
}

// ContentRegionAvail 返回当前容器里光标右侧/下方剩余的空间。
func (ui *UI) ContentRegionAvail() Rect {
	l := ui.layout
	return Rect{
		X: l.cursor.X,
		Y: l.cursor.Y,
		W: ui.availableWidth(),
		H: max(0, l.region.Y+l.region.H-l.cursor.Y),
	}
}

func (ui *UI) availableWidth() float32 {
	l := ui.layout
	return max(0, l.region.X+l.region.W-l.cursor.X)
}

func (ui *UI) availableHeight() float32 {
	l := ui.layout
	return max(0, l.region.Y+l.region.H-l.cursor.Y)
}

// itemRect 计算下一个 item 的矩形。w/h <= 0 时分别取剩余宽度 / 默认行高。
func (ui *UI) itemRect(w, h float32) Rect {
	l := &ui.layout
	if h <= 0 {
		h = ui.style.FrameHeight
	}
	if w <= 0 {
		w = ui.availableWidth()
	}
	// 上一个 item 已经结束了当前行，这里开始新行。
	if l.cursor.Y != l.lineY {
		l.lineY = l.cursor.Y
		l.lineH = 0
		l.lastX = l.region.X + l.indent
	}
	return Rect{l.cursor.X, l.cursor.Y, w, h}
}

// ItemRect 申请下一个控件的矩形（逻辑单位），用于自定义控件。
//
// w <= 0 表示占满当前行剩余宽度，h <= 0 表示用 Style.FrameHeight。
// 拿到矩形后自己绘制、自己 Hit，最后返回给 Add（Add 会推进布局），
// 或者手工调用 AdvanceItem。
func (ui *UI) ItemRect(w, h float32) Rect { return ui.itemRect(w, h) }

// AdvanceItem 推进布局光标。只有不走 ui.Add 的函数式控件才需要手动调用。
func (ui *UI) AdvanceItem(r Rect) { ui.advanceItem(r) }

// advanceItem 在 item 画完之后推进光标。
func (ui *UI) advanceItem(r Rect) {
	l := &ui.layout
	if !r.Empty() {
		l.lineH = max(l.lineH, r.H)
		l.lastX = r.X + r.W
	}
	l.cursor.Y = l.lineY + l.lineH + ui.style.ItemSpacing.Y
	l.cursor.X = l.region.X + l.indent
}

// SameLine 让下一个 item 接着当前行往右排。
func (ui *UI) SameLine() {
	l := &ui.layout
	l.cursor.Y = l.lineY
	l.cursor.X = max(l.lastX, l.region.X+l.indent) + ui.style.ItemSpacing.X
}

// NewLine 结束当前行，光标回到下一行行首。
func (ui *UI) NewLine() {
	l := &ui.layout
	l.cursor.Y = l.lineY + l.lineH + ui.style.ItemSpacing.Y
	l.lineY = l.cursor.Y
	l.lineH = 0
	l.cursor.X = l.region.X + l.indent
}

// Spacing 空出一行间距。
func (ui *UI) Spacing() {
	ui.NewLine()
	ui.layout.cursor.Y += ui.style.ItemSpacing.Y
	ui.layout.lineY = ui.layout.cursor.Y
}

// Dummy 占位一个 size 大小的空间（不绘制）。
func (ui *UI) Dummy(size Vec2) Rect {
	r := ui.itemRect(size.X, size.Y)
	ui.advanceItem(r)
	return r
}

// Indent 增加左缩进。
func (ui *UI) Indent(d float32) {
	ui.layout.indent += d
	ui.layout.cursor.X += d
}

// Unindent 减少左缩进。
func (ui *UI) Unindent(d float32) {
	l := &ui.layout
	l.indent = max(0, l.indent-d)
	l.cursor.X = max(l.region.X+l.indent, l.cursor.X-d)
}

// BeginChild 打开一个子区域：独立的布局光标、裁剪和滚动位置，用 EndChild 关闭。
// size 为 0 的分量表示占满当前剩余空间。
func (ui *UI) BeginChild(id string, size Vec2) {
	cid := ui.ID(id)
	w := size.X
	if w <= 0 {
		w = ui.availableWidth()
	}
	h := size.Y
	if h <= 0 {
		h = ui.availableHeight()
	}
	r := ui.itemRect(w, h)

	var scroll float32
	if ui.ctx != nil {
		scroll = ui.ctx.scrollOf(cid)
		if ui.ctx.activeID == 0 && ui.ctx.Input.mouseIn(r) {
			scroll -= ui.ctx.Input.Wheel.Y * childScrollSpeed
		}
		scroll = max(0, scroll)
	}

	ui.layouts = append(ui.layouts, ui.layout)
	ui.children = append(ui.children, childState{id: cid, rect: r, scroll: scroll})
	ui.layout = layout{
		region: Rect{r.X, r.Y, r.W, r.H},
		cursor: Vec2{r.X, r.Y - scroll},
		lineY:  r.Y - scroll,
		lastX:  r.X,
	}
	ui.PushClip(r)
}

// EndChild 关闭最近一个 BeginChild 打开的子区域，并把父布局推进一个子区域的高度。
func (ui *UI) EndChild() {
	n := len(ui.children)
	if n == 0 {
		return
	}
	ch := ui.children[n-1]
	ui.children = ui.children[:n-1]

	// 内容高度（子区域坐标，未滚动）
	contentH := max(0, (ui.layout.cursor.Y-ui.style.ItemSpacing.Y)-(ch.rect.Y-ch.scroll))
	maxScroll := max(0, contentH-ch.rect.H)
	scroll := min(max(ch.scroll, 0), maxScroll)

	ui.drawScrollbar(ch.rect, scroll, maxScroll)
	ui.PopClip()

	m := len(ui.layouts)
	ui.layout = ui.layouts[m-1]
	ui.layouts = ui.layouts[:m-1]
	ui.advanceItem(ch.rect)

	if ui.ctx != nil {
		ui.ctx.setScroll(ch.id, scroll)
	}
}

// drawScrollbar 在子区域右侧画一个滚动条指示器。
func (ui *UI) drawScrollbar(r Rect, scroll, maxScroll float32) {
	if maxScroll <= 0 || r.H < 32 || maxScroll < r.H*0.02 {
		return
	}
	const (
		barW   = 4
		margin = 4
	)
	trackH := r.H - margin*2
	contentH := maxScroll + r.H
	handleH := max(20, trackH*r.H/contentH)
	y := r.Y + margin + (trackH-handleH)*(scroll/maxScroll)
	ui.FillRoundRect(Rect{r.X + r.W - barW - 2, y, barW, handleH}, barW/2, ui.style.Scrollbar)
}
