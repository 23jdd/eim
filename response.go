package eim

type Rect struct {
	X, Y, W, H float32
}
type Response struct {
	Id   int64
	Area Rect
}
