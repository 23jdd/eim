package eim

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Options 是 App 的基础配置，零值字段表示使用默认值。
type Options struct {
	Title      string    // 窗口标题
	Width      int       // 逻辑宽度，<=0 用 640
	Height     int       // 逻辑高度，<=0 用 480
	Resizable  bool      // 允许拖拽调整窗口大小
	Fullscreen bool      // 全屏启动
	TPS        int       // 逻辑帧率，<=0 用 ebiten 默认值
	Background Color     // 每帧清屏色；零值（透明）表示不清屏
	Font       text.Face // 默认字体；nil 用内置字体
	Scale      float32   // 强制 DPI 缩放；0 表示跟随系统
	CloseOnEsc bool      // 按 Esc 退出（有输入框聚焦时只取消聚焦）
}

// withDefaults 补齐零值。
func (o Options) withDefaults() Options {
	if o.Width <= 0 {
		o.Width = 640
	}
	if o.Height <= 0 {
		o.Height = 480
	}
	return o
}

type App struct {
	view   func(ui *UI)
	opts   Options
	ctx    Context
	mu     sync.Mutex
	closed bool
}

// New 创建一个使用默认配置的 App，等价于 NewOptions(Options{})。
func New() *App { return NewOptions(Options{}) }

// NewOptions 用给定配置创建 App。
//
//	app := eim.NewOptions(eim.Options{
//		Title:      "my app",
//		Width:      800,
//		Height:     600,
//		Background: eim.Hex("#1e1f22"),
//		CloseOnEsc: true,
//	})
//	if err := app.View(draw).Run(); err != nil { panic(err) }
func NewOptions(opts Options) *App {
	app := &App{opts: opts}
	if opts.Scale > 0 {
		app.ctx.scaleOverride = opts.Scale
		app.ctx.scale = opts.Scale
	}
	return app
}

// Options 返回当前配置。
func (app *App) Options() Options { return app.opts }

func (app *App) Update() error {
	if app.closed {
		return ebiten.Termination
	}
	if app.opts.CloseOnEsc && app.ctx.focusID == 0 && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	app.ctx.Update()
	return nil
}

func (app *App) Draw(screen *ebiten.Image) {
	// Background 是一帧的底色，省得每个 view 都自己铺一遍。
	if app.opts.Background.A > 0 {
		screen.Fill(app.opts.Background)
	}
	if app.view == nil {
		return
	}

	app.ctx.beginFrame()
	ui := newUI(screen, &app.ctx)
	if app.opts.Font != nil {
		ui.SetFont(app.opts.Font)
	}
	app.view(ui)
	ui.flush()
}

// Layout 返回 Draw 收到的离屏图像尺寸（像素）。
//
// outside 是窗口大小，单位是设备无关像素(DIP)；乘上 DeviceScaleFactor 后离屏图像
// 与窗口的物理像素 1:1，文字在高 DPI 屏上才是清晰的。
// 这个坐标空间就是 draw 层的坐标空间：ebiten 会据此把光标位置换算回来，
// 所以 Context.Update 里不要再乘 DeviceScaleFactor。
func (app *App) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	scale := ebiten.Monitor().DeviceScaleFactor()
	return int(math.Round(float64(outsideWidth) * scale)), int(math.Round(float64(outsideHeight) * scale))
}

// Run 应用 Options 里的窗口配置并进入主循环。
func (app *App) Run() error {
	o := app.opts.withDefaults()
	app.opts = o

	if o.Title != "" {
		ebiten.SetWindowTitle(o.Title)
	}
	ebiten.SetWindowSize(o.Width, o.Height)
	ebiten.SetWindowResizable(o.Resizable)
	if o.TPS > 0 {
		ebiten.SetTPS(o.TPS)
	}
	if o.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	return ebiten.RunGame(app)
}

func (app *App) Close() {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.closed = true
}

func (app *App) View(view func(ui *UI)) *App {
	app.view = view
	return app
}
