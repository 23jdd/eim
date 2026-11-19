# eim 使用指南

`eim` 是一个 Dear ImGui 风格的**即时模式** UI 库，底层用 [Ebiten](https://ebitengine.org)。

- 每帧重新描述界面，不维护控件树
- 布局是「光标 + 顺序堆叠」，`SameLine` 换行
- 事件由矩形命中测试驱动：hover / pressed / clicked / active / double-click
- 坐标一律是**逻辑单位**，DPI（125% / 150% / 200%）自动处理

---

## 目录

1. [安装与最小示例](#安装与最小示例)
2. [核心概念](#核心概念)
3. [布局](#布局)
4. [内置控件](#内置控件)
5. [文本与字体](#文本与字体)
6. [交互与事件](#交互与事件)
7. [自定义控件](#自定义控件)
8. [图片](#图片)
9. [绘制图元](#绘制图元)
10. [样式](#样式)
11. [完整示例](#完整示例)
12. [已知限制](#已知限制)

---

## 安装与最小示例

```bash
go get github.com/23jdd/eim
```

```go
package main

import (
	"fmt"

	"github.com/23jdd/eim"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(480, 320)
	ebiten.SetWindowTitle("hello eim")
	if err := eim.New().View(draw).Run(); err != nil {
		panic(err)
	}
}

func draw(ui *eim.UI) {
	ui.FillRect(ui.ScreenBounds(), ui.Style().WindowBG)
	ui.SetRegion(ui.ScreenBounds().Inset(16))

	ui.Label("hello, eim")
	if ui.Button("click me") {
		fmt.Println("clicked")
	}
}
```

`New()` 返回 `*App`，`View(func(ui *UI))` 注册每帧回调，`Run()` 进入 Ebiten 主循环。需要退出时调 `App.Close()`。

### 启动配置（Options）

窗口和运行时参数用 `NewOptions` 一次配好：

```go
app := eim.NewOptions(eim.Options{
	Title:      "my app",
	Width:      800,
	Height:     600,
	Background: eim.Hex("#1e1f22"),
	CloseOnEsc: true,
})
if err := app.View(draw).Run(); err != nil {
	panic(err)
}
```

零值即默认值，`New()` 等价于 `NewOptions(Options{})`；窗口设置是在 `Run()` 时才应用的。

| 字段 | 默认 | 说明 |
|---|---|---|
| `Title` | `""` | 窗口标题，空则不设置 |
| `Width` / `Height` | 640 / 480 | **逻辑**尺寸（DIP），高 DPI 下物理像素更大 |
| `Resizable` | `false` | 允许拖拽调整窗口 |
| `Fullscreen` | `false` | 全屏启动 |
| `TPS` | Ebiten 默认 | 逻辑帧率 |
| `Background` | 透明 | 每帧清屏色；透明表示不清屏，需要自己铺底 |
| `Font` | `nil` | 默认字体，等价于每帧 `ui.SetFont` |
| `Scale` | 0 | 强制 DPI 缩放；0 = 跟随系统 |
| `CloseOnEsc` | `false` | Esc 退出；有输入框聚焦时只取消聚焦 |

设了 `Background` 之后，`draw` 里就不必再 `FillRect(ScreenBounds(), ...)` 了。`app.Options()` 可以读回当前配置（`Run()` 之后宽高已补成默认值）。

---

## 核心概念

### 每帧重建

`draw` 每帧被调用一次，UI 是「用完即弃」的。**状态要放在外面**：

```go
var clicks int

func draw(ui *eim.UI) {
	if ui.Button("click") {
		clicks++
	}
	ui.Label(fmt.Sprintf("clicks = %d", clicks))
}
```

### 坐标与 DPI

- 所有坐标、尺寸、字号都是**逻辑单位**（100% 缩放下的像素）
- `App.Layout` 返回物理像素，`draw.go` 在落笔时统一乘 DPI 缩放
- 字体按 `16 * dpi` 栅格化，`TextSize` 再除回逻辑值

所以同一份代码在 100% 和 150% 屏上**物理尺寸一致且清晰**，你不需要写任何 `* scale`。

### 帧与输入

`App.Update` 采集输入 → `App.Draw` 里跑 `View` → 帧末画 tooltip。
一帧内 Ebiten 可能多次调用 `Draw`，一次性的按键/点击事件只会被消费一次。

---

## 布局

### 内容区域

```go
ui.SetRegion(panel.Inset(16)) // 设置布局区域并重置光标
```

`SetRegion` 同时决定裁剪区域：超出区域的控件不会被命中，但**不会**被裁剪绘制（要裁剪用 `BeginChild` 或 `PushClip`）。

### 顺序堆叠

控件按调用顺序从上往下排，行距由 `Style.ItemSpacing` 决定。

```go
ui.Label("name")
ui.Label("age")     // 自动换到下一行
```

### 横向排列

```go
ui.Button("OK")
ui.SameLine()
ui.Button("Cancel")
```

`SameLine()` 让下一个控件回到当前行右侧；`NewLine()` 结束当前行。

### 间距与占位

| 方法 | 作用 |
|---|---|
| `Spacing()` | 空出一行间距 |
| `Dummy(Vec2{w, h})` | 占位，不绘制，返回矩形 |
| `Indent(d)` / `Unindent(d)` | 左右缩进 |

### 子区域（裁剪 + 滚动）

```go
ui.BeginChild("list", eim.Vec2{X: 0, Y: 160}) // 宽 0 = 占满，高 0 = 占满剩余
for i := 0; i < 100; i++ {
	ui.Selectable(fmt.Sprintf("row %d", i), sel == i)
}
ui.EndChild()
```

- 子区域有独立的布局光标、裁剪和滚动位置
- 鼠标在子区域内滚动滚轮即可滚动，内容超高时右侧自动出现滚动条
- 可以嵌套；`EndChild` 会把父布局推进一个子区域的高度

### 查询与手动定位

```go
r := ui.ContentRegionAvail() // 光标右侧/下方剩余空间
p := ui.CursorPos()          // 下一个控件的位置
ui.SetCursorPos(eim.Vec2{X: 100, Y: 40})
```

`ItemRect(w, h)` 申请一个矩形（`w <= 0` 占满剩余宽度，`h <= 0` 用 `FrameHeight`），`AdvanceItem(r)` 推进光标——主要给自定义控件用，详见[自定义控件](#自定义控件)。

---

## 内置控件

| 控件 | 签名 | 返回 |
|---|---|---|
| 按钮 | `Button(label)` | 本帧是否点击 |
| 复选 | `Checkbox(label, *bool)` | 是否改变 |
| 开关 | `Switch(label, *bool)` | 是否改变 |
| 单选 | `RadioButton(label, *int, value)` | 是否改变 |
| 列表行 | `Selectable(label, selected)` | 是否点击 |
| 浮点滑条 | `SliderFloat(label, *float64, min, max)` | 是否改变 |
| 整数滑条 | `SliderInt(label, *int, min, max)` | 是否改变 |
| 双端滑条 | `RangeSlider(label, *lo, *hi, min, max)` | 是否改变 |
| 拖拽数值 | `DragFloat(label, *float64, speed)` / `DragInt(label, *int, speed)` | 是否改变 |
| 旋钮 | `Knob(label, *float64, min, max)` | 是否改变 |
| 输入框 | `InputText(label, *string)` | 文本是否改变 |
| 进度条 | `ProgressBar(fraction, text)` | — |
| 颜色块 | `ColorButton(label, color, size)` | 是否点击 |
| 取色 | `ColorEdit(label, *Color)` | 是否改变 |
| 标签 | `Badge(text)` | — |
| 加载指示 | `Spinner()` | — |
| 折线图 | `PlotLines([]float64)` | — |
| 柱状图 | `PlotHistogram([]float64)` | — |
| 分隔线 | `Separator()` / `SeparatorText(label)` | — |
| 文本 | `Label(s)` / `TextColored(s, c)` / `TextDisabled(s)` / `TextWrapped(s)` / `BulletText(s)` | — |
| 键值行 | `Pair(label, value)` / `LabelText(label, value)` | — |

控件 ID 由 label 生成，**同一帧里不要出现两个相同 label**（循环里用 `PushID`，见下）。

---

## 文本与字体

### 排版辅助

```go
w, h := ui.TextSize("hello")    // 逻辑尺寸
lh := ui.TextHeight()           // 一行高度
ui.Text(x, y, "hello", c)       // (x,y) 是左上角
ui.TextCentered(r, "hello", c)  // 在矩形内居中，超宽自动裁剪
ui.TextLeft(r, "hello", 8, c)   // 左对齐 + 垂直居中
ui.TextRight(r, "hello", 8, c)  // 右对齐 + 垂直居中
ui.WrapText(s, maxW)            // 手动折行，返回行数组
```

### 中文 / 自定义字体

内置默认字体只有 ASCII（Go Regular）。要中文就在 `draw` 里设一次：

```go
var face text.Face

func main() {
	f, err := eim.LoadFontFile("C:/Windows/Fonts/msyh.ttc", 16)
	if err == nil {
		face = f
	}
	// ... Run()
}

func draw(ui *eim.UI) {
	if face != nil {
		ui.SetFont(face) // 注意：size 是逻辑像素，高 DPI 下不会自动放大
	}
}
```

- `LoadFontFile(path, size)` 支持 ttf / otf / ttc（取第一个字体）
- `LoadFont(io.Reader, size)` 从内存加载
- `ui.SetFont(nil)` 恢复默认
- 自己设的字体不会按 DPI 缩放，需要的话用 `16 * ui.Scale()` 作为 size

---

## 交互与事件

内置控件已经处理好了点击，直接用返回值即可。需要更细的交互（hover 换色、拖拽、双击）时用 `Hit`：

```go
r := ui.ItemRect(100, 30)
ev := ui.Hit(eim.Response{Id: ui.ID("card"), Area: r})

ev.Hovered()       // 鼠标在矩形上（且没有被别的控件抢走）
ev.Pressed()       // 本帧按下
ev.Released()      // 本帧松开
ev.Clicked()       // 按下和松开都在矩形内
ev.Active()        // 正被按住
ev.DoubleClicked() // 400ms 内第二次点击
```

规则：

- 同一帧里**第一个**命中的矩形拿到 hover，后面的控件不会抢
- 按住期间其他控件不能 hover（拖拽不会误触）
- 松开时鼠标不在矩形内 → 不算 click，但 `Active()` 在按住期间一直为真
- 被裁剪掉（`PushClip` / `BeginChild`）的区域不参与命中

### 焦点

```go
id := ui.ID("field")
if ui.Focused(id) { /* 键盘焦点在我这 */ }
ui.SetFocus(id)
ui.ClearFocus()
```

点击任何调用过 `Hit` 的控件都会把焦点切过去。

### Tooltip

```go
ui.Button("hover me")
if ui.ItemHovered() {
	ui.Tooltip("这是提示")
}
```

`ItemHovered()` 指「最近一个调用 `Hit` 的控件」。tooltip 在帧末统一绘制，永远盖在最上面。

### 直接读输入

```go
in := ui.Input() // 可能为 nil
in.Mouse         // 逻辑坐标
in.MouseDelta    // 本帧位移（拖拽用）
in.MouseDown[0]  // 左键按住
in.MousePressed[0] / in.MouseReleased[0]
in.Wheel         // 滚轮
in.Chars         // 本帧输入的字符
in.KeyPressed(ebiten.KeyBackspace)
```

---

## 自定义控件

自定义控件和内置控件平权：申请矩形 → 命中测试 → 绘制 → 交回矩形。

```go
type Chip struct {
	Label string
	On    bool
}

// Render 实现 eim.Widget
func (c *Chip) Render(ui *eim.UI) eim.Response {
	id := ui.ID(c.Label)
	tw, th := ui.TextSize(c.Label)

	// 1. 申请矩形
	r := ui.ItemRect(tw+20, th+8)

	// 2. 命中测试
	ev := ui.Hit(eim.Response{Id: id, Area: r})

	// 3. 绘制
	bg := ui.Style().FrameBG
	switch {
	case c.On:
		bg = ui.Style().Accent
	case ev.Hovered():
		bg = ui.Style().FrameBGHovered
	}
	ui.FillRoundRect(r, r.H/2, bg)
	ui.TextCentered(r, c.Label, ui.Style().Text)

	// 4. 状态
	if ev.Clicked() {
		c.On = !c.On
	}

	// 5. 交回矩形，Add 负责推进布局
	return eim.Response{Id: id, Area: r}
}
```

```go
chips := []*Chip{{Label: "alpha", On: true}, {Label: "beta"}}
for _, c := range chips {
	ui.Add(c)
	ui.SameLine()
}
```

### 约定

1. **`Render` 不要推进布局**，`Add` 会按返回的 `Area` 推进（自己推会双倍间距）。
2. **状态放结构体字段**，跨帧由你持有。
3. **ID 要稳定**：用 `ui.ID(label)`；循环里重复标签用 `ui.PushID(key)` / `ui.PopID()`。
4. 需要像素级绘制（位图、1 物理像素细线）时用 `ui.Scale()` 换算。

### 一次性控件

```go
ui.Add(eim.WidgetFunc(func(ui *eim.UI) eim.Response {
	r := ui.ItemRect(0, 0)
	ev := ui.Hit(eim.Response{Id: ui.ID("x"), Area: r})
	if ev.Hovered() {
		ui.Tooltip("inline")
	}
	return eim.Response{Id: ui.ID("x"), Area: r}
}))
```

### 自定义控件可用的 API

| 类别 | API |
|---|---|
| 布局 | `ItemRect` `AdvanceItem` `SameLine` `NewLine` `Spacing` `Dummy` `Indent` `BeginChild/EndChild` `ContentRegionAvail` `CursorPos/SetCursorPos` |
| 事件 | `Hit` `Events.*` `ItemHovered` `Focused/SetFocus/ClearFocus` `ID/PushID/PopID` `Input` |
| 绘制 | 全部图元 + `Text` `TextCentered` `TextLeft/TextRight` `TextSize` `TextHeight` `WrapText` `Image` `DrawImage` `PushClip/PopClip` |
| 样式 | `Style()`（可直接改字段，跨帧保留）`Scale()` |

---

## 图片

```go
ui.Image("assets/logo.png", eim.Rect{X: 0, Y: 0, W: 64, H: 64})
```

- 路径相对于进程工作目录，支持 png / jpeg / gif
- **按路径缓存**，同一路径只读一次盘
- 加载失败时什么也不画；要错误处理或预加载用 `LoadImageFile`

```go
w, h := ui.ImageSize("assets/logo.png") // 原始像素尺寸，失败返回 0,0

img, err := eim.LoadImageFile("assets/logo.png")
if err != nil {
	log.Fatal(err)
}
ui.DrawImage(img, r) // 画已加载的图片（也支持子图、渲染目标）
```

按宽高比排版：

```go
w, h := ui.ImageSize(path)
if h > 0 {
	width := 200
	r := ui.ItemRect(float32(width), float32(width)*h/w)
	ui.Image(path, r)
	ui.AdvanceItem(r)
}
```

---

## 绘制图元

```go
ui.FillRect(r, c)
ui.StrokeRect(r, thickness, c)                 // 内描边
ui.FillRoundRect(r, rounding, c)
ui.StrokeRoundRect(r, rounding, thickness, c)  // 内描边
ui.Line(x0, y0, x1, y1, thickness, c)
ui.FillCircle(cx, cy, radius, c)
ui.StrokeCircle(cx, cy, radius, thickness, c)
ui.FillTriangle(a, b, c, col)
ui.FillPolygon([]eim.Vec2{...}, c)
ui.StrokePolygon([]eim.Vec2{...}, thickness, c)
ui.StrokePolyline([]eim.Vec2{...}, thickness, c)
```

颜色：

```go
eim.Hex("#4c8bf5")        // 支持 #rgb / #rrggbb / #rrggbbaa
eim.RGB(76, 139, 245)
eim.RGBA(76, 139, 245, 128)
eim.FromColor(color.White)
eim.ColorWhite / eim.ColorBlack / eim.ColorTransparent
c.WithAlpha(0x80)
```

几何：

```go
r.Inset(8)          // 内缩
r.Intersect(o)      // 交集
r.Empty()
r.Center()
```

裁剪：

```go
ui.PushClip(r)
ui.FillRect(big, c)
ui.PopClip()
```

---

## 样式

```go
st := ui.Style()          // *Style，修改跨帧保留
st.Accent = eim.Hex("#34a853")

ui.SetStyle(eim.DefaultStyle()) // 整体重置
```

| 字段 | 说明 |
|---|---|
| `Text` `TextMuted` `Disabled` | 文字颜色 |
| `WindowBG` `FrameBG` | 面板 / 控件底色 |
| `FrameBGHovered` `FrameBGActive` | hover / 按下 |
| `Border` `Accent` `CheckMark` `Scrollbar` | 边框、强调色等 |
| `FramePadding` | 控件内边距 |
| `ItemSpacing` | 控件间距 |
| `FrameRounding` | 圆角 |
| `FrameHeight` | 单行控件高度 |
| `IndentSpacing` | 缩进 |
| `MinButtonWidth` | 按钮最小宽度 |

样式尺寸都是逻辑单位，DPI 由库处理。

---

## 完整示例

`examples/main.go` 是一个两列布局的完整 demo：

```bash
go run ./examples
```

包含：按钮行、开关、复选框、滑条、进度条、输入框、可选列表（带 tooltip）、单选、双端滑条、拖拽、旋钮、Spinner、Badge、折线图，以及一个用公开 API 写的自定义 `chip` 控件。

---

## 已知限制

- **没有 popup / menu / combo / modal**：这些需要 overlay 层（z-order + 点击外部关闭），目前没做
- **没有 Table / TabBar / CollapsingHeader**：需要列模型和持久展开状态
- `InputText` 只支持在末尾输入/删除，没有光标移动、选中、剪贴板
- Tooltip 每帧只有一个，没有延迟/淡入
- 图片缓存不监听文件变化，改了图要重启
- 子区域滚轮方向是「向下滚 → 内容上移」，如果手感相反可以在 `layout.go` 里翻符号
- 没有键盘导航（Tab 切换焦点）

需要其中某一项可以按这个顺序加：overlay 层 → Combo/Menu → TabBar → Table。
