package bubble

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/nfnt/resize"
)

//go:embed bubble_mask.png
var bubbleMaskBytes []byte
var Mask *image.Alpha

var StdDrawer *Drawer

func init() {
	img, err := png.Decode(bytes.NewReader(bubbleMaskBytes))
	if err != nil {
		panic(err)
	}

	alphaImage := image.NewAlpha(img.Bounds())
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			_, _, _, a := img.At(x, y).RGBA()

			alphaImage.SetAlpha(x, y, color.Alpha{A: uint8(a)})
		}
	}

	Mask = alphaImage

	StdDrawer = New(nil, Mask)
}

type Drawer struct {
	Base draw.Drawer
	Mask *image.Alpha
}

func New(base draw.Drawer, mask *image.Alpha) *Drawer {
	if mask == nil {
		panic("Mask is nil.")
	}

	return &Drawer{
		Base: base,
		Mask: mask,
	}
}

var transparent = color.RGBA{0, 0, 0, 0}

func (d Drawer) Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
	// Draw the original one if we haven't already
	if d.Base != nil {
		d.Base.Draw(dst, r, src, sp)
	}

	// Setup our mask
	maskBounds := d.Mask.Bounds()
	maskRatio := maskBounds.Dx() / maskBounds.Dy()
	maskW := r.Dx()
	maskH := maskW / maskRatio
	fmt.Println(maskH)
	alphaMask := image.NewAlpha(r)
	resizedMask := resize.Resize(uint(maskW), uint(maskH), d.Mask, resize.Bicubic)
	draw.Draw(alphaMask, image.Rect(0, 0, maskW, maskH), resizedMask, sp, draw.Src)

	// Mask everything out
	w, h := maskW, maskH
	for x := range w + 1 {
		for y := range h + 1 {
			alpha := alphaMask.AlphaAt(x, y)
			col := dst.At(x, y)

			if alpha.A == 255 {
				col = transparent
				dst.Set(x, y, col)
			}
		}
	}
}
