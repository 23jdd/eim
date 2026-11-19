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

// Image 从文件路径加载图片，缩放绘制到 r（逻辑单位）。
//
// 路径相对于进程工作目录；同一个路径只读一次盘，之后走缓存。
// 加载失败时什么也不画——需要错误处理就用 LoadImageFile 自己加载，
// 再用 DrawImage 画。
func (ui *UI) Image(path string, r Rect) {
	var (
		img *ebiten.Image
		err error
	)
	if ui.ctx != nil {
		img, err = ui.ctx.image(path)
	} else {
		img, err = LoadImageFile(path)
	}
	if err != nil || img == nil {
		return
	}
	ui.DrawImage(img, r)
}

// ImageSize 返回图片的像素尺寸，加载失败返回 (0, 0)。
// 常用于按原始宽高比排版。
func (ui *UI) ImageSize(path string) (w, h float32) {
	var (
		img *ebiten.Image
		err error
	)
	if ui.ctx != nil {
		img, err = ui.ctx.image(path)
	} else {
		img, err = LoadImageFile(path)
	}
	if err != nil || img == nil || img.Bounds().Empty() {
		return 0, 0
	}
	b := img.Bounds()
	return float32(b.Dx()), float32(b.Dy())
}

// DrawImage 把已经加载好的图片缩放绘制到 r（逻辑单位）。
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
