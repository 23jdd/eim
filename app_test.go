package eim

import "testing"

func TestNewOptions(t *testing.T) {
	app := NewOptions(Options{
		Title:  "demo",
		Width:  800,
		Height: 600,
		Scale:  2,
	})

	got := app.Options()
	if got.Title != "demo" || got.Width != 800 || got.Height != 600 {
		t.Fatalf("options: %+v", got)
	}
	if app.ctx.scaleOverride != 2 || app.ctx.scale != 2 {
		t.Fatalf("scale: override=%v scale=%v", app.ctx.scaleOverride, app.ctx.scale)
	}
	if ui := newUI(nil, &app.ctx); ui.Scale() != 2 {
		t.Fatalf("ui scale: got %v; want 2", ui.Scale())
	}
}

func TestOptionsDefaults(t *testing.T) {
	o := Options{}.withDefaults()
	if o.Width != 640 || o.Height != 480 {
		t.Fatalf("defaults: got %dx%d; want 640x480", o.Width, o.Height)
	}

	o = Options{Width: 320, Height: 200}.withDefaults()
	if o.Width != 320 || o.Height != 200 {
		t.Fatalf("explicit size changed: got %dx%d", o.Width, o.Height)
	}

	o = Options{Width: 320}.withDefaults()
	if o.Width != 320 || o.Height != 480 {
		t.Fatalf("partial: got %dx%d; want 320x480", o.Width, o.Height)
	}
}

func TestSetScaleReset(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)

	ui.SetScale(2)
	if ctx.scaleOverride != 2 {
		t.Fatalf("override: got %v", ctx.scaleOverride)
	}

	ui.SetScale(0) // 恢复跟随系统
	if ctx.scaleOverride != 0 {
		t.Fatalf("override should reset: got %v", ctx.scaleOverride)
	}
}
