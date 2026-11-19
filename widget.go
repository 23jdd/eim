package eim

// Widget 是自定义控件的接口。
//
// Render 只负责三件事：申请矩形、命中测试、绘制，然后返回自己的矩形。
//
//	func (w *MyWidget) Render(ui *eim.UI) Response {
//		id := ui.ID(w.label)
//		r := ui.ItemRect(w.width, w.height)          // 1. 申请位置
//		ev := ui.Hit(Response{Id: id, Area: r})      // 2. 事件
//		ui.FillRoundRect(r, 4, ui.Style().FrameBG)   // 3. 绘制
//		ui.TextCentered(r, w.label, ui.Style().Text)
//		return Response{Id: id, Area: r}             // 4. 交回矩形
//	}
//
// 不要在 Render 里推进布局，ui.Add 会按返回的 Area 推进。
// 需要跨帧状态就把字段放在实现类型上，Add 时传指针即可。
type Widget interface {
	Render(ui *UI) Response
}

// WidgetFunc 让普通函数也能当 Widget 用，适合一次性的小控件：
//
//	ui.Add(eim.WidgetFunc(func(ui *eim.UI) eim.Response {
//		r := ui.ItemRect(0, 0)
//		return eim.Response{Id: ui.ID("x"), Area: r}
//	}))
type WidgetFunc func(ui *UI) Response

// Render 实现 Widget。
func (f WidgetFunc) Render(ui *UI) Response { return f(ui) }
