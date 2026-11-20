package eim

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeTestPNG 在临时目录里写一张 4x3 的图。
func writeTestPNG(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "img.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(0, 0, color.RGBA{R: 0xff, A: 0xff})
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadImageFile(t *testing.T) {
	path := writeTestPNG(t)
	img, err := LoadImageFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Fatalf("bounds: got %v", b)
	}
}

func TestImageSizeCache(t *testing.T) {
	path := writeTestPNG(t)
	ctx := &Context{}
	ui := newUI(nil, ctx)

	if w, h := ui.ImageSize(path); w != 4 || h != 3 {
		t.Fatalf("ImageSize: got %vx%v; want 4x3", w, h)
	}
	// 同一路径只加载一次
	a, err := ctx.image(path)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ctx.image(path)
	if a != b {
		t.Fatalf("image should be cached by path")
	}
}

func TestImageLayout(t *testing.T) {
	path := writeTestPNG(t)
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 300, 300})

	// 只给宽：按原图比例算高（4x3 -> 40x30）
	r := ui.Image(path, Vec2{X: 40})
	if r.W != 40 || r.H != 30 {
		t.Fatalf("aspect ratio: got %vx%v; want 40x30", r.W, r.H)
	}
	if got, want := ui.CursorPos().Y, 30+ui.style.ItemSpacing.Y; got != want {
		t.Fatalf("layout advance: got %v; want %v", got, want)
	}

	// 只给高：按比例算宽
	r = ui.Image(path, Vec2{Y: 30})
	if r.W != 40 || r.H != 30 {
		t.Fatalf("aspect ratio by height: got %vx%v; want 40x30", r.W, r.H)
	}

	// 都不给：原图尺寸
	ui.SetRegion(Rect{0, 0, 300, 300})
	r = ui.Image(path, Vec2{})
	if r.W != 4 || r.H != 3 {
		t.Fatalf("natural size: got %vx%v; want 4x3", r.W, r.H)
	}

	// 都给：拉伸
	r = ui.Image(path, Vec2{X: 20, Y: 50})
	if r.W != 20 || r.H != 50 {
		t.Fatalf("stretch: got %vx%v; want 20x50", r.W, r.H)
	}
}

func TestImageMissing(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	ui.SetRegion(Rect{0, 0, 300, 300})
	missing := filepath.Join(t.TempDir(), "nope.png")

	if w, h := ui.ImageSize(missing); w != 0 || h != 0 {
		t.Fatalf("missing image size: got %vx%v", w, h)
	}
	// 加载失败不推进布局
	if r := ui.Image(missing, Vec2{X: 40}); !r.Empty() {
		t.Fatalf("missing image should return empty rect: %v", r)
	}
	if got := ui.CursorPos(); got != (Vec2{}) {
		t.Fatalf("layout should not advance: %v", got)
	}
	ui.DrawImageFile(missing, Rect{W: 10, H: 10}) // 不 panic

	if _, err := ctx.image(missing); err == nil {
		t.Fatalf("missing image should return an error")
	}
}
