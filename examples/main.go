package main

import (
	"fmt"

	"github.com/23jdd/eim"
)

var (
	clicks = 0
	check  = true
	value  = 0.35
	name   = "eim"
	on     = true
	mode   = 0
	ival   = 40
	lo, hi = 0.2, 0.8
	drag   = 12.5
	knob   = 0.6
	sel    = 1
	plot   = []float64{3, 7, 4, 9, 6, 11, 8, 13, 7, 10, 5, 9}
)

func main() {
	// 需要中文时：载入一个带 CJK 的字体，然后传给 Options.Font。
	// face, _ := eim.LoadFontFile("C:/Windows/Fonts/msyh.ttc", 16)

	app := eim.NewOptions(eim.Options{
		Title:      "eim: widgets",
		Width:      600,
		Height:     560,
		Background: eim.Hex("#1e1f22"),
		CloseOnEsc: true,
	})
	if err := app.View(draw).Run(); err != nil {
		panic(err)
	}
}

func draw(ui *eim.UI) {
	panel := ui.ScreenBounds().Inset(20)
	ui.FillRoundRect(panel, 10, eim.Hex("#2b2d30"))
	ui.StrokeRoundRect(panel, 10, 1, ui.Style().Border)

	content := panel.Inset(16)
	ui.SetRegion(content)

	ui.TextColored("eim: layout / widgets / events", ui.Style().Accent)
	ui.Separator()

	// 两列：两个 BeginChild 并排
	colW := (content.W - 16) / 2
	ui.BeginChild("left", eim.Vec2{X: colW, Y: 0})
	leftColumn(ui)
	ui.EndChild()

	ui.SameLine()
	ui.BeginChild("right", eim.Vec2{X: 0, Y: 0})
	rightColumn(ui)
	ui.EndChild()
}

func leftColumn(ui *eim.UI) {
	if ui.Button("Click me") {
		clicks++
	}
	ui.SameLine()
	if ui.Button("Reset") {
		clicks = 0
		value = 0.35
	}
	ui.SameLine()
	ui.Label(fmt.Sprintf("clicks = %d", clicks))

	ui.Switch("switch", &on)
	ui.Checkbox("checkbox", &check)
	ui.SliderFloat("slider", &value, 0, 1)
	ui.ProgressBar(float32(value), fmt.Sprintf("%.0f%%", value*100))
	ui.InputText("name", &name)

	ui.Separator()
	ui.Label("selectable:")
	ui.BeginChild("list", eim.Vec2{X: 0, Y: 140})
	for i := 0; i < 6; i++ {
		if ui.Selectable(fmt.Sprintf("item %d", i), sel == i) {
			sel = i
		}
		if ui.ItemHovered() {
			ui.Tooltip("click to select")
		}
	}
	ui.EndChild()
}

func rightColumn(ui *eim.UI) {
	ui.RadioButton("mode a", &mode, 0)
	ui.RadioButton("mode b", &mode, 1)
	ui.SliderInt("int", &ival, 0, 100)
	ui.RangeSlider("range", &lo, &hi, 0, 1)
	ui.DragFloat("drag", &drag, 0.1)

	ui.Spacing()
	ui.Knob("knob", &knob, 0, 1)
	if ui.ItemHovered() {
		ui.Tooltip(fmt.Sprintf("knob = %.2f", knob))
	}
	ui.SameLine()
	ui.Spinner()
	ui.SameLine()
	ui.Badge("new")
	ui.SameLine()
	ui.Badge("beta")

	ui.Spacing()
	ui.PlotLines(plot)

	ui.Spacing()
	ui.Label("custom widget:")
	for _, c := range chips {
		ui.Add(c)
		ui.SameLine()
	}
}

// chip 是一个自定义控件：不依赖任何包内私有函数，只用导出的 API。
type chip struct {
	label string
	on    bool
}

var chips = []*chip{{label: "alpha", on: true}, {label: "beta"}, {label: "gamma"}}

func (c *chip) Render(ui *eim.UI) eim.Response {
	id := ui.ID(c.label)
	tw, th := ui.TextSize(c.label)

	// 1. 申请矩形：宽 = 文字 + 左右 padding，高 = 文字 + 上下 padding
	r := ui.ItemRect(tw+20, th+8)

	// 2. 命中测试
	ev := ui.Hit(eim.Response{Id: id, Area: r})

	// 3. 绘制
	bg := ui.Style().FrameBG
	switch {
	case c.on:
		bg = ui.Style().Accent
	case ev.Hovered():
		bg = ui.Style().FrameBGHovered
	}
	ui.FillRoundRect(r, r.H/2, bg)
	ui.TextCentered(r, c.label, ui.Style().Text)

	// 4. 状态
	if ev.Clicked() {
		c.on = !c.on
	}

	// 5. 交回矩形，Add 会推进布局
	return eim.Response{Id: id, Area: r}
}
