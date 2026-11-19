package eim

// Style 控制控件的外观和间距。
//
// 所有尺寸都是逻辑单位（100% DPI 下的像素），与 DPI 无关。
type Style struct {
	Text      Color // 正文
	TextMuted Color // 次要文字
	Disabled  Color // 禁用态

	WindowBG Color // 窗口/面板底色
	FrameBG  Color // 输入框、滑条等控件底色

	FrameBGHovered Color
	FrameBGActive  Color

	Border    Color
	Accent    Color // 高亮/进度条填充
	CheckMark Color
	Scrollbar Color

	FramePadding   Vec2 // 控件内部边距
	ItemSpacing    Vec2 // 控件之间的间距
	FrameRounding  float32
	FrameHeight    float32
	IndentSpacing  float32
	MinButtonWidth float32
}

// defaultStyle 只构造一次，避免每帧解析颜色字符串。
var defaultStyle = Style{
	Text:      Hex("#e6e6e6"),
	TextMuted: Hex("#9aa0a6"),
	Disabled:  Hex("#6b6f76"),

	WindowBG: Hex("#1e1f22"),
	FrameBG:  Hex("#35373b"),

	FrameBGHovered: Hex("#404247"),
	FrameBGActive:  Hex("#4a4d52"),

	Border:    Hex("#4a4d52"),
	Accent:    Hex("#4c8bf5"),
	CheckMark: Hex("#ffffff"),
	Scrollbar: Hex("#4a4d52"),

	FramePadding:   Vec2{8, 4},
	ItemSpacing:    Vec2{8, 6},
	FrameRounding:  4,
	FrameHeight:    26,
	IndentSpacing:  16,
	MinButtonWidth: 80,
}

// DefaultStyle 返回默认的暗色主题。
func DefaultStyle() Style { return defaultStyle }

// styleFor 返回 Context 上持久化的样式（没有则用默认样式初始化）。
func (c *Context) styleFor() *Style {
	if c.style == nil {
		s := DefaultStyle()
		c.style = &s
	}
	return c.style
}

// Style 返回当前样式。修改会跨帧保留。
func (ui *UI) Style() *Style { return ui.style }

// SetStyle 整体替换样式。
func (ui *UI) SetStyle(s Style) {
	if ui.ctx != nil {
		*ui.ctx.styleFor() = s
		return
	}
	*ui.style = s
}

// textHeight 返回一行文字的逻辑高度。
func (ui *UI) textHeight() float32 {
	if _, h := ui.TextSize("Ag"); h > 0 {
		return h
	}
	return ui.style.FrameHeight
}

// TextHeight 返回一行文字的高度（逻辑单位），自定义控件排版用。
func (ui *UI) TextHeight() float32 { return ui.textHeight() }

// TextLeft 在矩形内左对齐、垂直居中绘制文字。
func (ui *UI) TextLeft(r Rect, s string, padX float32, c Color) {
	ui.textLeft(r, s, padX, c)
}

// TextRight 在矩形内右对齐、垂直居中绘制文字。
func (ui *UI) TextRight(r Rect, s string, padX float32, c Color) {
	w, h := ui.TextSize(s)
	ui.Text(r.X+r.W-padX-w, r.Y+(r.H-h)/2, s, c)
}

// labelIn 在矩形内居中绘制文字，文字超宽时裁掉。
func (ui *UI) labelIn(r Rect, s string, c Color) {
	tw, th := ui.TextSize(s)
	clipped := tw > r.W
	if clipped {
		ui.PushClip(r)
	}
	ui.Text(r.X+(r.W-tw)/2, r.Y+(r.H-th)/2, s, c)
	if clipped {
		ui.PopClip()
	}
}

// textLeft 在矩形内左对齐、垂直居中绘制文字。
func (ui *UI) textLeft(r Rect, s string, padX float32, c Color) {
	_, th := ui.TextSize(s)
	ui.Text(r.X+padX, r.Y+(r.H-th)/2, s, c)
}
