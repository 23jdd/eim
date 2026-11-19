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
	// 没有 screen 时绘制是 no-op，但不应 panic
	ui.Image(path, Rect{X: 0, Y: 0, W: 40, H: 30})
}

func TestImageMissing(t *testing.T) {
	ctx := &Context{}
	ui := newUI(nil, ctx)
	missing := filepath.Join(t.TempDir(), "nope.png")

	if w, h := ui.ImageSize(missing); w != 0 || h != 0 {
		t.Fatalf("missing image size: got %vx%v", w, h)
	}
	ui.Image(missing, Rect{W: 10, H: 10}) // 不 panic

	if _, err := ctx.image(missing); err == nil {
		t.Fatalf("missing image should return an error")
	}
}
