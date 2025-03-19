package gifdrawer

import (
	"image"
	"image/color"
	"image/draw"
)

// GifDrawer is a carbon copy of the Floyd-Steinberg implimentation in the
// go standard library, but it doens't perform dithering on transparent
// pixels. Instead, it attmpts to preserve them by just setting to alpha
// to zero and going onto the next pixel.
type GifDrawer struct {
	dr draw.Drawer
}

func New() *GifDrawer {
	return &GifDrawer{draw.FloydSteinberg}
}

var transparent = color.RGBA{0, 0, 0, 0}

func (gf GifDrawer) Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
	// Let the normal drawer handle the dificult stuff
	// todo: don't draw the transparency on until after...
	gf.dr.Draw(dst, r, src, sp)

	// Copy the transparency of the src image to the dst image
	w, h := r.Dx(), r.Dy()
	for x := range w + 1 {
		for y := range h + 1 {
			_, _, _, srcAlpha := src.At(x, y).RGBA()
			dstCol := dst.At(x, y)

			// We need to update the whole color to one
			// in the palette.
			if srcAlpha == 0 {
				dstCol = transparent
			}

			dst.Set(x, y, dstCol)
		}
	}
}
