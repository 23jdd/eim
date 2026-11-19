package eim

import "github.com/hajimehoshi/ebiten/v2"

// UI 是一帧的即时模式 UI 上下文。
//
// 坐标约定：UI 里所有坐标和尺寸都是「逻辑单位」（100% DPI 下的像素），
// 与 DPI 无关。draw.go 在落笔时统一乘 ui.scale 换算成屏幕像素，
// 所以写控件的人不需要关心 DPI。
type UI struct {
	screen *ebiten.Image
	ctx    *Context
	style  *Style
	scale  float32 // DPI 缩放，1 = 100%

	draw drawState // 绘制状态，见 draw.go

	layout   layout
	layouts  []layout
	children []childState
	ids      []int64

	tip    string // 待绘制的 tooltip，帧末统一画
	lastID int64  // 最近一次 Hit 的控件，用于 ItemHovered
}

func newUI(screen *ebiten.Image, ctx *Context) *UI {
	ui := &UI{screen: screen, ctx: ctx}

	scale := float32(1)
	if ctx != nil && ctx.scale > 0 {
		scale = ctx.scale
	}
	ui.scale = scale

	if ctx != nil {
		ui.style = ctx.styleFor()
	} else {
		s := DefaultStyle()
		ui.style = &s
	}

	if screen != nil {
		ui.SetRegion(ui.ScreenBounds())
	}
	return ui
}

// Scale 返回当前 DPI 缩放（1 = 100%）。一般不需要用到，
// 只有需要按物理像素对齐的自定义绘制（比如 1px 发丝线、位图）才用得上。
func (ui *UI) Scale() float32 { return ui.scaleOr1() }

// SetScale 强制 DPI 缩放，传 0 恢复跟随系统。
// 也可以用 Options.Scale 在启动时指定。
func (ui *UI) SetScale(s float32) {
	if s < 0 {
		s = 0
	}
	ui.scale = s
	if ui.ctx != nil {
		ui.ctx.scaleOverride = s
		if s > 0 {
			ui.ctx.scale = s
		}
	}
}

func (ui *UI) scaleOr1() float32 {
	if ui.scale <= 0 {
		return 1
	}
	return ui.scale
}

// px 把逻辑尺寸换算成屏幕像素。
func (ui *UI) px(v float32) float32 { return v * ui.scaleOr1() }

// pr 把逻辑矩形换算成屏幕像素矩形。
func (ui *UI) pr(r Rect) Rect {
	s := ui.scaleOr1()
	if s == 1 {
		return r
	}
	return Rect{r.X * s, r.Y * s, r.W * s, r.H * s}
}

// Add 渲染一个自定义 Widget，并按它返回的矩形推进布局。
//
// Widget.Render 只负责绘制并返回自己的矩形，不要自己推进布局；Add 会推进。
// 交互用 ui.Hit 在 Render 内部完成。
func (ui *UI) Add(w Widget) Response {
	r := w.Render(ui)
	if !r.Area.Empty() {
		ui.advanceItem(r.Area)
	}
	return r
}

// Input 返回本帧的输入快照（只读）。自定义控件用它实现拖拽、文本输入等：
//
//	in := ui.Input()
//	if in != nil && in.MousePressed[0] { ... }
//	for _, ch := range in.Chars { ... }
//
// ctx 为 nil 时返回 nil。
func (ui *UI) Input() *Input {
	if ui.ctx == nil {
		return nil
	}
	return &ui.ctx.Input
}

// ItemHovered 报告刚刚绘制的控件是否被悬停，用于紧跟控件加 tooltip：
//
//	ui.Button("...")
//	if ui.ItemHovered() { ui.Tooltip("...") }
func (ui *UI) ItemHovered() bool {
	return ui.ctx != nil && ui.lastID != 0 && ui.ctx.hotID == ui.lastID
}

// flush 画帧末内容（目前是 tooltip），由 App.Draw 在 view 之后调用。
func (ui *UI) flush() {
	if ui.tip == "" {
		return
	}
	text := ui.tip
	ui.tip = ""

	// tooltip 要盖在所有东西上面，临时清掉裁剪栈。
	saved := ui.draw.clips
	ui.draw.clips = nil
	ui.draw.cached = false
	defer func() {
		ui.draw.clips = saved
		ui.draw.cached = false
	}()

	const pad = 6
	tw, th := ui.TextSize(text)
	w, h := tw+2*pad, th+2*pad
	b := ui.ScreenBounds()

	x, y := b.X+4, b.Y+4
	if ui.ctx != nil {
		x = ui.ctx.Input.Mouse.X + 14
		y = ui.ctx.Input.Mouse.Y + 18
		if y+h > b.Y+b.H {
			y = ui.ctx.Input.Mouse.Y - h - 10
		}
	}
	x = min(max(x, b.X+4), max(b.X+4, b.X+b.W-w-4))
	y = max(y, b.Y+4)

	r := Rect{x, y, w, h}
	ui.FillRoundRect(r, 4, ui.style.FrameBG)
	ui.StrokeRoundRect(r, 4, 1, ui.style.Border)
	ui.Text(x+pad, y+pad, text, ui.style.Text)
}
