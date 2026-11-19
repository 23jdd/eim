package eim

// 本文件实现 UI 的基础图元绘制：矩形、圆角矩形、线、圆、多边形、图片和文字。
//
// 约定：
//   - 坐标为屏幕像素坐标，原点在左上角，y 轴向下。
//   - 颜色为非预乘(straight) alpha 的 8 位 RGBA，见 Color。
//   - 所有绘制都受 PushClip/PopClip 压入的裁剪区域限制。
//   - 边框和线宽都向矩形「内侧」绘制，相邻控件不会互相压盖。

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

// drawState 是 UI 的绘制状态。
type drawState struct {
	clips  []Rect
	font   text.Face
	target *ebiten.Image // 裁剪后的绘制目标，见 (*UI).target
	cached bool
}

// Color 是直通(非预乘) alpha 的 8 位 RGBA 颜色。
// 它实现了 color.Color，可以直接传给 ebiten / vector 的绘图函数。
type Color struct {
	R, G, B, A uint8
}

// RGB 构造一个不透明的颜色。
func RGB(r, g, b uint8) Color { return Color{r, g, b, 0xff} }

// RGBA 构造一个带 alpha 的颜色。
func RGBA(r, g, b, a uint8) Color { return Color{r, g, b, a} }

// RGBA 实现 color.Color，返回值是预乘 alpha 的 16 位分量。
func (c Color) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	r *= uint32(c.A)
	r /= 0xff

	g = uint32(c.G)
	g |= g << 8
	g *= uint32(c.A)
	g /= 0xff

	b = uint32(c.B)
	b |= b << 8
	b *= uint32(c.A)
	b /= 0xff

	a = uint32(c.A)
	a |= a << 8
	return r, g, b, a
}

// WithAlpha 返回替换了 alpha 通道的颜色。
func (c Color) WithAlpha(a uint8) Color {
	c.A = a
	return c
}

// Hex 解析 "#rgb"、"#rrggbb"、"#rrggbbaa"（'#' 可省略），解析失败返回全透明。
func Hex(s string) Color {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		// #rgb => #rrggbb
		s = s[0:1] + s[0:1] + s[1:2] + s[1:2] + s[2:3] + s[2:3]
	}
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return Color{}
	}
	switch len(s) {
	case 6:
		return Color{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
	case 8:
		return Color{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)}
	}
	return Color{}
}

// FromColor 把任意 color.Color 转换成 Color。
func FromColor(c color.Color) Color {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return Color{n.R, n.G, n.B, n.A}
}

// 常用颜色。
var (
	ColorTransparent = Color{}
	ColorBlack       = Color{0x00, 0x00, 0x00, 0xff}
	ColorWhite       = Color{0xff, 0xff, 0xff, 0xff}
)

// Vec2 是二维点/向量。
type Vec2 struct{ X, Y float32 }

// Empty 报告矩形是否为空（宽或高不为正）。
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Inset 返回向内收缩 d 的矩形；d 为负时向外扩张。
func (r Rect) Inset(d float32) Rect {
	return Rect{X: r.X + d, Y: r.Y + d, W: r.W - 2*d, H: r.H - 2*d}
}

// Intersect 返回两个矩形的交集，无交集时返回空矩形。
func (r Rect) Intersect(o Rect) Rect {
	x0 := max(r.X, o.X)
	y0 := max(r.Y, o.Y)
	x1 := min(r.X+r.W, o.X+o.W)
	y1 := min(r.Y+r.H, o.Y+o.H)
	if x1 <= x0 || y1 <= y0 {
		return Rect{X: x0, Y: y0}
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

// Center 返回矩形中心点。
func (r Rect) Center() Vec2 { return Vec2{r.X + r.W/2, r.Y + r.H/2} }

// ScreenBounds 返回屏幕（画布）矩形，逻辑单位。
func (ui *UI) ScreenBounds() Rect {
	if ui.screen == nil {
		return Rect{}
	}
	b := ui.screen.Bounds()
	s := ui.scaleOr1()
	return Rect{float32(b.Min.X) / s, float32(b.Min.Y) / s, float32(b.Dx()) / s, float32(b.Dy()) / s}
}

// PushClip 把后续绘制裁剪到 r（与当前裁剪区域求交），需与 PopClip 成对使用。
func (ui *UI) PushClip(r Rect) {
	if n := len(ui.draw.clips); n > 0 {
		r = r.Intersect(ui.draw.clips[n-1])
	}
	ui.draw.clips = append(ui.draw.clips, r)
	ui.draw.cached = false
}

// PopClip 弹出最近一次 PushClip 压入的裁剪区域。
func (ui *UI) PopClip() {
	if n := len(ui.draw.clips); n > 0 {
		ui.draw.clips = ui.draw.clips[:n-1]
		ui.draw.cached = false
	}
}

// ClipRect 返回当前裁剪区域（没有 PushClip 时就是当前布局区域）。
func (ui *UI) ClipRect() Rect {
	if n := len(ui.draw.clips); n > 0 {
		return ui.draw.clips[n-1]
	}
	if r := ui.layout.region; !r.Empty() {
		return r
	}
	return ui.ScreenBounds()
}

// Visible 报告矩形 r 是否与当前裁剪区域相交，可用于跳过不可见的绘制。
func (ui *UI) Visible(r Rect) bool {
	return !r.Intersect(ui.ClipRect()).Empty()
}

// target 返回当前绘制目标：整屏，或裁剪后的子图。
//
// ebiten 的子图作为绘制目标时沿用原图坐标系，只是把绘制裁到子图范围，
// 所以调用方不需要做坐标偏移。
func (ui *UI) target() *ebiten.Image {
	if ui.draw.cached {
		return ui.draw.target
	}
	ui.draw.cached = true
	ui.draw.target = nil
	if ui.screen == nil {
		return nil
	}

	dst := ui.screen
	if n := len(ui.draw.clips); n > 0 {
		r := ui.draw.clips[n-1]
		pr := ui.pr(r)
		ir := image.Rect(
			int(math.Floor(float64(pr.X))),
			int(math.Floor(float64(pr.Y))),
			int(math.Ceil(float64(pr.X+pr.W))),
			int(math.Ceil(float64(pr.Y+pr.H))),
		).Intersect(dst.Bounds())
		if ir.Empty() {
			return nil
		}
		sub, ok := dst.SubImage(ir).(*ebiten.Image)
		if !ok {
			return nil
		}
		dst = sub
	}
	ui.draw.target = dst
	return dst
}

// FillRect 用纯色填充矩形（不抗锯齿，适合像素对齐的 UI 面板）。
func (ui *UI) FillRect(r Rect, c Color) {
	if r.Empty() || c.A == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	pr := ui.pr(r)
	vector.FillRect(dst, pr.X, pr.Y, pr.W, pr.H, c, false)
}

// StrokeRect 在矩形内侧描边，线宽不会超出 r 的范围。
func (ui *UI) StrokeRect(r Rect, thickness float32, c Color) {
	if r.Empty() || thickness <= 0 || c.A == 0 {
		return
	}
	t := min(thickness, min(r.W, r.H)/2)
	ui.FillRect(Rect{r.X, r.Y, r.W, t}, c)
	ui.FillRect(Rect{r.X, r.Y + r.H - t, r.W, t}, c)
	ui.FillRect(Rect{r.X, r.Y + t, t, r.H - 2*t}, c)
	ui.FillRect(Rect{r.X + r.W - t, r.Y + t, t, r.H - 2*t}, c)
}

// FillRoundRect 填充圆角矩形，rounding 为圆角半径（会被限制在短边的一半以内）。
func (ui *UI) FillRoundRect(r Rect, rounding float32, c Color) {
	if rounding <= 0 {
		ui.FillRect(r, c)
		return
	}
	if r.Empty() || c.A == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	vector.FillPath(dst, roundRectPath(ui.pr(r), ui.px(rounding)), nil, pathOptions(c))
}

// StrokeRoundRect 在圆角矩形内侧描边。
func (ui *UI) StrokeRoundRect(r Rect, rounding, thickness float32, c Color) {
	if r.Empty() || thickness <= 0 || c.A == 0 {
		return
	}
	if rounding <= 0 {
		ui.StrokeRect(r, thickness, c)
		return
	}
	inner := r.Inset(thickness / 2)
	if inner.Empty() {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	stroke := &vector.StrokeOptions{
		Width:    ui.px(thickness),
		LineJoin: vector.LineJoinRound,
	}
	vector.StrokePath(dst, roundRectPath(ui.pr(inner), ui.px(max(rounding-thickness/2, 0))), stroke, pathOptions(c))
}

// roundRectPath 构造圆角矩形的路径。
func roundRectPath(r Rect, rounding float32) *vector.Path {
	rad := min(rounding, min(r.W, r.H)/2)
	if rad <= 0 {
		p := &vector.Path{}
		p.MoveTo(r.X, r.Y)
		p.LineTo(r.X+r.W, r.Y)
		p.LineTo(r.X+r.W, r.Y+r.H)
		p.LineTo(r.X, r.Y+r.H)
		p.Close()
		return p
	}

	const (
		halfPi    = math.Pi / 2
		threeHalf = 3 * math.Pi / 2
	)
	x0, y0 := r.X, r.Y
	x1, y1 := r.X+r.W, r.Y+r.H

	p := &vector.Path{}
	p.MoveTo(x0+rad, y0)
	p.LineTo(x1-rad, y0)
	p.Arc(x1-rad, y0+rad, rad, -halfPi, 0, vector.Clockwise)
	p.LineTo(x1, y1-rad)
	p.Arc(x1-rad, y1-rad, rad, 0, halfPi, vector.Clockwise)
	p.LineTo(x0+rad, y1)
	p.Arc(x0+rad, y1-rad, rad, halfPi, math.Pi, vector.Clockwise)
	p.LineTo(x0, y0+rad)
	p.Arc(x0+rad, y0+rad, rad, math.Pi, threeHalf, vector.Clockwise)
	p.Close()
	return p
}

// Line 画一条线段 (x0,y0)-(x1,y1)。
func (ui *UI) Line(x0, y0, x1, y1, thickness float32, c Color) {
	if thickness <= 0 || c.A == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	vector.StrokeLine(dst, ui.px(x0), ui.px(y0), ui.px(x1), ui.px(y1), ui.px(thickness), c, true)
}

// FillCircle 填充圆。
func (ui *UI) FillCircle(cx, cy, radius float32, c Color) {
	if radius <= 0 || c.A == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	vector.FillCircle(dst, ui.px(cx), ui.px(cy), ui.px(radius), c, true)
}

// StrokeCircle 在圆内侧描边。
func (ui *UI) StrokeCircle(cx, cy, radius, thickness float32, c Color) {
	if radius <= 0 || thickness <= 0 || c.A == 0 {
		return
	}
	inner := radius - thickness/2
	if inner <= 0 {
		ui.FillCircle(cx, cy, radius, c)
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	vector.StrokeCircle(dst, ui.px(cx), ui.px(cy), ui.px(inner), ui.px(thickness), c, true)
}

// FillTriangle 填充三角形。
func (ui *UI) FillTriangle(a, b, c Vec2, col Color) {
	ui.FillPolygon([]Vec2{a, b, c}, col)
}

// FillPolygon 填充简单多边形（凹多边形也可，使用非零环绕规则）。
func (ui *UI) FillPolygon(pts []Vec2, c Color) {
	if len(pts) < 3 || c.A == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	vector.FillPath(dst, ui.polyPath(pts, true), nil, pathOptions(c))
}

// StrokePolygon 沿多边形的各条边描边（闭合）。
func (ui *UI) StrokePolygon(pts []Vec2, thickness float32, c Color) {
	ui.strokePoly(pts, thickness, c, true)
}

// StrokePolyline 沿折线描边（不闭合）。
func (ui *UI) StrokePolyline(pts []Vec2, thickness float32, c Color) {
	ui.strokePoly(pts, thickness, c, false)
}

func (ui *UI) strokePoly(pts []Vec2, thickness float32, c Color, closed bool) {
	if len(pts) < 2 || thickness <= 0 || c.A == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	stroke := &vector.StrokeOptions{
		Width:    ui.px(thickness),
		LineJoin: vector.LineJoinRound,
		LineCap:  vector.LineCapRound,
	}
	vector.StrokePath(dst, ui.polyPath(pts, closed), stroke, pathOptions(c))
}

// polyPath 构造多边形路径（已换算成屏幕像素）。
func (ui *UI) polyPath(pts []Vec2, closed bool) *vector.Path {
	p := &vector.Path{}
	p.MoveTo(ui.px(pts[0].X), ui.px(pts[0].Y))
	for _, pt := range pts[1:] {
		p.LineTo(ui.px(pt.X), ui.px(pt.Y))
	}
	if closed {
		p.Close()
	}
	return p
}

// pathOptions 把颜色转换成填充/描边路径的选项。
func pathOptions(c Color) *vector.DrawPathOptions {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	return op
}

// SetFont 设置文本字体，传 nil 恢复内置默认字体。
func (ui *UI) SetFont(f text.Face) { ui.draw.font = f }

// Font 返回当前字体。没有单独设置时，返回按 DPI 缩放后的内置字体。
func (ui *UI) Font() text.Face {
	if ui.draw.font != nil {
		return ui.draw.font
	}
	size := float64(baseFontSize) * float64(ui.scale)
	if size <= 0 {
		size = baseFontSize
	}
	return defaultFaceAt(size)
}

// baseFontSize 是 100% DPI 下的默认字号（逻辑像素）。
const baseFontSize = 16

var (
	defaultSourceOnce sync.Once
	defaultSourceVal  *text.GoTextFaceSource

	faceMu    sync.Mutex
	faceCache = map[float64]text.Face{}
)

// defaultSource 解析内置的 Go Regular 字体，只解析一次。
func defaultSource() *text.GoTextFaceSource {
	defaultSourceOnce.Do(func() {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		if err != nil {
			return
		}
		defaultSourceVal = src
	})
	return defaultSourceVal
}

// defaultFaceAt 返回指定像素大小的内置字体，按大小缓存。
func defaultFaceAt(size float64) text.Face {
	src := defaultSource()
	if src == nil {
		return nil
	}
	faceMu.Lock()
	defer faceMu.Unlock()
	if f, ok := faceCache[size]; ok {
		return f
	}
	f := &text.GoTextFace{Source: src, Size: size}
	faceCache[size] = f
	return f
}

// Text 以 (x, y) 为左上角绘制文本，'\n' 换行。
func (ui *UI) Text(x, y float32, s string, c Color) {
	if s == "" || c.A == 0 {
		return
	}
	face := ui.Font()
	if face == nil {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	m := face.Metrics()
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(ui.px(x)), float64(ui.px(y)))
	op.LineSpacing = m.HAscent + m.HDescent + m.HLineGap
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

// TextSize 返回文本在屏幕上的像素尺寸（与 Text 的排版一致）。
func (ui *UI) TextSize(s string) (w, h float32) {
	face := ui.Font()
	if face == nil || s == "" {
		return 0, 0
	}
	m := face.Metrics()
	w64, h64 := text.Measure(s, face, m.HAscent+m.HDescent+m.HLineGap)
	k := ui.scaleOr1()
	return float32(w64) / k, float32(h64) / k
}

// TextCentered 在 r 中居中绘制文本，文字超宽时裁掉。
func (ui *UI) TextCentered(r Rect, s string, c Color) {
	w, h := ui.TextSize(s)
	if w > r.W {
		ui.PushClip(r)
		defer ui.PopClip()
	}
	ui.Text(r.X+(r.W-w)/2, r.Y+(r.H-h)/2, s, c)
}

// LoadFont 从 ttf/otf 数据加载字体，size 为像素大小。
func LoadFont(r io.Reader, size float64) (text.Face, error) {
	src, err := text.NewGoTextFaceSource(r)
	if err != nil {
		return nil, err
	}
	return &text.GoTextFace{Source: src, Size: size}, nil
}

// LoadFontFile 从字体文件加载字体，支持 ttf/otf 和 ttc 字体集合（取第一个字体）。
// 需要中文时可以用系统字体，例如 Windows 的 "C:/Windows/Fonts/msyh.ttc"。
func LoadFontFile(path string, size float64) (text.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) >= 4 && string(data[:4]) == "ttcf" {
		srcs, err := text.NewGoTextFaceSourcesFromCollection(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if len(srcs) == 0 {
			return nil, fmt.Errorf("eim: 字体集合为空: %s", path)
		}
		return &text.GoTextFace{Source: srcs[0], Size: size}, nil
	}
	return LoadFont(bytes.NewReader(data), size)
}
