package eim

// 事件系统：把 ebiten 的原始输入变成「某个矩形本帧被 hover / 按住 / 点击」。
//
// 用法（控件内部或自定义 Widget）：
//
//	ev := ui.Hit(Response{Id: ui.ID("ok"), Area: r})
//	if ev.Clicked() { ... }
//
// Hit 同时负责 hover 抢占和 active 仲裁：同一帧里第一个命中的矩形拿到 hover，
// 按住期间其他矩形不会被 hover，松开时只有「按下和松开都在自己身上」才算 click。

import (
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input 是一帧的输入快照，由 Context.Update 从 ebiten 采集。
type Input struct {
	Mouse         Vec2
	MouseDelta    Vec2
	MouseDown     [3]bool
	MousePressed  [3]bool
	MouseReleased [3]bool
	Wheel         Vec2
	Chars         []rune

	keyBuf []ebiten.Key
}

// KeyPressed 报告 key 是否在这一帧刚刚按下。
func (in *Input) KeyPressed(key ebiten.Key) bool {
	for _, k := range in.keyBuf {
		if k == key {
			return true
		}
	}
	return false
}

func (in *Input) mouseIn(r Rect) bool {
	return r.W > 0 && r.H > 0 &&
		in.Mouse.X >= r.X && in.Mouse.X < r.X+r.W &&
		in.Mouse.Y >= r.Y && in.Mouse.Y < r.Y+r.H
}

// clearOneShot 清掉「只应该生效一次」的事件，避免一帧内多次 Draw 时重复触发。
func (in *Input) clearOneShot() {
	in.MousePressed = [3]bool{}
	in.MouseReleased = [3]bool{}
	in.Wheel = Vec2{}
	in.Chars = in.Chars[:0]
	in.keyBuf = in.keyBuf[:0]
}

// Context 保存跨帧的 UI 状态：输入快照、hover/active/focus、滚动位置。
// 一个 App 持有一个 Context：Update 里刷新输入，Draw 里被控件消费。
type Context struct {
	Input Input

	hotID    int64 // 本帧被 hover 的控件
	activeID int64 // 正被按住的控件
	focusID  int64 // 键盘焦点

	scroll map[int64]float32
	images map[string]imageEntry

	scale         float32 // 当前 DPI 缩放，Update 里计算
	scaleOverride float32 // 非 0 时强制使用该缩放
	style         *Style  // 持久化的样式（逻辑单位）

	lastClickID   int64
	lastClickTime time.Time

	newFrame bool
	primed   bool
}

// Update 采集本帧输入，由 App.Update 调用。
func (c *Context) Update() {
	in := &c.Input

	// 屏幕是「物理像素」尺寸（见 App.Layout），样式和字号按 dpi 放大，
	// 这样高 DPI 屏上既清晰、物理尺寸又和 100% 时一致。
	scale := c.scaleOverride
	if scale <= 0 {
		scale = float32(ebiten.Monitor().DeviceScaleFactor())
	}
	if scale <= 0 {
		scale = 1
	}
	c.scale = scale

	// ebiten 的光标是屏幕像素坐标（已含 DPI 和 letterbox 换算），
	// 而 UI 里用的是逻辑单位，所以这里除回 scale。
	x, y := ebiten.CursorPositionF()
	pos := Vec2{float32(x) / scale, float32(y) / scale}
	if c.primed {
		in.MouseDelta = Vec2{pos.X - in.Mouse.X, pos.Y - in.Mouse.Y}
	}
	in.Mouse = pos

	for i := range in.MouseDown {
		b := ebiten.MouseButton(i)
		in.MouseDown[i] = ebiten.IsMouseButtonPressed(b)
		in.MousePressed[i] = inpututil.IsMouseButtonJustPressed(b)
		in.MouseReleased[i] = inpututil.IsMouseButtonJustReleased(b)
	}

	wx, wy := ebiten.Wheel()
	in.Wheel = Vec2{float32(wx), float32(wy)}
	in.Chars = ebiten.AppendInputChars(in.Chars[:0])
	in.keyBuf = inpututil.AppendJustPressedKeys(in.keyBuf[:0])

	c.primed = true
	c.newFrame = true
}

// beginFrame 每帧开始时调用：处理一帧内多次 Draw，并重置 hover。
func (c *Context) beginFrame() {
	if !c.newFrame {
		c.Input.clearOneShot()
	}
	c.newFrame = false
	c.hotID = 0
}

func (c *Context) scrollOf(id int64) float32 {
	return c.scroll[id]
}

func (c *Context) setScroll(id int64, v float32) {
	if c.scroll == nil {
		c.scroll = make(map[int64]float32)
	}
	c.scroll[id] = v
}

// Events 描述某个矩形在本帧的交互状态。所有方法对 nil 安全。
type Events struct {
	Id   int64
	Area Rect

	hovered  bool
	pressed  bool
	released bool
	clicked  bool
	active   bool
	doubled  bool
}

// Hovered 报告鼠标是否在本帧悬停在这个矩形上。
func (e *Events) Hovered() bool { return e != nil && e.hovered }

// Pressed 报告鼠标是否在本帧按在这个矩形上。
func (e *Events) Pressed() bool { return e != nil && e.pressed }

// Released 报告本帧是否在按住状态下松开了鼠标（不一定构成 click）。
func (e *Events) Released() bool { return e != nil && e.released }

// Clicked 报告本帧是否完成了一次点击（按下和松开都在矩形内）。
func (e *Events) Clicked() bool { return e != nil && e.clicked }

// Active 报告这个矩形是否正被按住。
func (e *Events) Active() bool { return e != nil && e.active }

// DoubleClicked 报告本帧是否完成了一次双击。
func (e *Events) DoubleClicked() bool { return e != nil && e.doubled }

// Hit 对响应矩形做命中测试，返回本帧的交互事件。
//
// 这是即时模式的核心：不要缓存 Events，每帧重新调用。
func (ui *UI) Hit(r Response) *Events {
	ev := &Events{Id: r.Id, Area: r.Area}
	ui.lastID = r.Id
	if ui.ctx == nil || r.Area.Empty() {
		return ev
	}
	if r.Id == 0 {
		// 没给 ID 的自定义控件：用矩形本身凑一个，布局稳定时也是稳定的。
		r.Id = int64(hashString(rectKey(r.Area)))
		ev.Id = r.Id
	}

	c := ui.ctx
	in := &c.Input

	claimed := c.hotID != 0 && c.hotID != r.Id
	blocked := c.activeID != 0 && c.activeID != r.Id
	inside := ui.Visible(r.Area) && in.mouseIn(r.Area.Intersect(ui.ClipRect()))

	if !claimed && !blocked && inside {
		ev.hovered = true
		c.hotID = r.Id
		if in.MousePressed[0] {
			c.activeID = r.Id
			c.focusID = r.Id
			ev.pressed = true
		}
	}

	if c.activeID == r.Id {
		ev.active = in.MouseDown[0]
		if in.MouseReleased[0] {
			ev.released = true
			ev.clicked = ev.hovered
			c.activeID = 0
			if ev.clicked {
				now := time.Now()
				if c.lastClickID == r.Id && now.Sub(c.lastClickTime) < 400*time.Millisecond {
					ev.doubled = true
				}
				c.lastClickID = r.Id
				c.lastClickTime = now
			}
		}
	}
	return ev
}

// Focused 报告某个 ID 是否持有键盘焦点。
func (ui *UI) Focused(id int64) bool { return ui.ctx != nil && ui.ctx.focusID == id }

// SetFocus 把键盘焦点交给某个 ID。
func (ui *UI) SetFocus(id int64) {
	if ui.ctx != nil {
		ui.ctx.focusID = id
	}
}

// ClearFocus 清除键盘焦点。
func (ui *UI) ClearFocus() {
	if ui.ctx != nil {
		ui.ctx.focusID = 0
	}
}

// ID 由标签生成稳定的控件 ID。同一个标签在不同 PushID 作用域下得到不同 ID。
func (ui *UI) ID(label string) int64 {
	v := int64(hashString(label))
	if n := len(ui.ids); n > 0 {
		v ^= ui.ids[n-1]
	}
	if v == 0 {
		v = 1
	}
	return v
}

// PushID 压入一个 ID 作用域，用于循环里重复的标签。
func (ui *UI) PushID(label string) {
	ui.ids = append(ui.ids, ui.ID(label))
}

// PopID 弹出最近一个 ID 作用域。
func (ui *UI) PopID() {
	if n := len(ui.ids); n > 0 {
		ui.ids = ui.ids[:n-1]
	}
}

// hashString 是 FNV-1a 64。
func hashString(s string) uint64 {
	const (
		offset = 14695981039346656037
		prime  = 1099511628211
	)
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

func rectKey(r Rect) string {
	return strconv.Itoa(int(r.X)) + "," + strconv.Itoa(int(r.Y)) + "," +
		strconv.Itoa(int(r.W)) + "," + strconv.Itoa(int(r.H))
}
