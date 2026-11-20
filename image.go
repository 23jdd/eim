package eim

import (
	"io"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// LoadImageFile 从文件加载图片（支持 png/jpeg/gif）。
// 一般不用自己调，ui.Image 会按路径缓存；需要预加载或处理错误时用它。
func LoadImageFile(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	return img, err
}

// LoadImage 从 reader 加载图片（png/jpeg/gif）。
func LoadImage(r io.Reader) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromReader(r)
	return img, err
}

type imageEntry struct {
	img *ebiten.Image
	err error
}

// image 按路径缓存加载结果（失败也缓存，避免每帧读盘）。
func (c *Context) image(path string) (*ebiten.Image, error) {
	if e, ok := c.images[path]; ok {
		return e.img, e.err
	}
	img, err := LoadImageFile(path)
	if c.images == nil {
		c.images = make(map[string]imageEntry)
	}
	c.images[path] = imageEntry{img: img, err: err}
	return img, err
}

func (ui *UI) imageFor(path string) (*ebiten.Image, error) {
	if ui.ctx != nil {
		return ui.ctx.image(path)
	}
	return LoadImageFile(path)
}

// Image 在布局光标处绘制图片，并推进布局。
//
// size 是逻辑尺寸：
//   - 都为 0：用原图像素尺寸（1 像素 = 1 逻辑单位）
//   - 只给宽：按原图宽高比算高；只给高：按比例算宽
//   - 都给：拉伸到该尺寸
//
// 返回实际占用的矩形；加载失败时返回空矩形且不推进布局。
// 路径相对于进程工作目录，同一个路径只读一次盘。
func (ui *UI) Image(path string, size Vec2) Rect {
	img, err := ui.imageFor(path)
	if err != nil || img == nil {
		return Rect{}
	}
	b := img.Bounds()
	iw, ih := float32(b.Dx()), float32(b.Dy())
	if iw <= 0 || ih <= 0 {
		return Rect{}
	}

	w, h := size.X, size.Y
	switch {
	case w <= 0 && h <= 0:
		w, h = iw, ih
	case w <= 0:
		w = h * iw / ih
	case h <= 0:
		h = w * ih / iw
	}

	r := ui.itemRect(w, h)
	ui.DrawImage(img, r)
	ui.advanceItem(r)
	return r
}

// ImageSize 返回图片的像素尺寸，加载失败返回 (0, 0)。
// 原图尺寸可以直接用作 Image 的 size（1 像素 = 1 逻辑单位）。
func (ui *UI) ImageSize(path string) (w, h float32) {
	img, err := ui.imageFor(path)
	if err != nil || img == nil || img.Bounds().Empty() {
		return 0, 0
	}
	b := img.Bounds()
	return float32(b.Dx()), float32(b.Dy())
}

// DrawImageFile 从文件加载图片并绘制到指定矩形（绝对定位，不参与布局）。
// 布局里请用 Image。
func (ui *UI) DrawImageFile(path string, r Rect) {
	img, err := ui.imageFor(path)
	if err != nil || img == nil {
		return
	}
	ui.DrawImage(img, r)
}

// DrawImage 把已经加载好的图片缩放绘制到 r（逻辑单位，绝对定位）。
func (ui *UI) DrawImage(img *ebiten.Image, r Rect) {
	if img == nil || r.Empty() {
		return
	}
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return
	}
	dst := ui.target()
	if dst == nil {
		return
	}
	pr := ui.pr(r)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(pr.W)/float64(b.Dx()), float64(pr.H)/float64(b.Dy()))
	op.GeoM.Translate(float64(pr.X), float64(pr.Y))
	dst.DrawImage(img, op)
}
