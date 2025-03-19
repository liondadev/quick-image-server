package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/ericpauley/go-quantize/quantize"
	"github.com/nfnt/resize"
)

//go:embed bubble_mask.png
var bubbleMaskBytes []byte
var bubbleMaskImg *image.Image

func init() {
	img, err := png.Decode(bytes.NewReader(bubbleMaskBytes))
	if err != nil {
		panic(err)
	}

	bubbleMaskImg = &img
}

// normalizeImage takes a normal image.Image and turns into
// an object that impliments draw.Image by creating an object
// that impliments draw.Image and drawing src over top of it.
// This is also guarenteed to return an image with rgba values
// of 0 <= rgba <= 255, unlike the jpeg library for some fucking
// reason.
func normalizeImage(img image.Image) draw.Image {
	drawImg := image.NewRGBA(img.Bounds())                          // create a new image that impliments the draw.Imgage
	draw.Draw(drawImg, img.Bounds(), img, image.Point{}, draw.Over) // draw the image over the new one

	return drawImg
}

// MakeBubbleImage creates one of those discord bubble images with the speech bubble over an image.
func (s *Server) MakeBubbleImage(mime string, original io.Reader) (image.Image, error) {
	var src image.Image

	switch mime {
	case "image/jpeg":
		dec, err := jpeg.Decode(original)
		if err != nil {
			return nil, fmt.Errorf("decode jpeg: %w", err)
		}
		src = dec
	case "image/png":
		dec, err := png.Decode(original)
		if err != nil {
			return nil, fmt.Errorf("decode png: %w", err)
		}
		src = dec
	default:
		return nil, fmt.Errorf("mime type '%s' can't be used to create bubble images", mime)
	}

	// The PNG image library already implimemts draw.Image on all
	// the returned images, but the JPEG library doesn't.
	dst := normalizeImage(src)

	mb := (*bubbleMaskImg).Bounds()
	mh := mb.Dy()

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// Resize the bubble mask image to the same size as the image
	// we're targetting. This allows us to then use mask.At to check the transparency
	// and do some manual masking.
	mask := image.NewRGBA(image.Rect(0, 0, w, h))
	resizedBaskMask := resize.Resize(uint(w), uint(mh), *bubbleMaskImg, resize.Lanczos3)
	draw.Draw(mask, image.Rect(0, 0, w, mh), resizedBaskMask, image.Point{}, draw.Over)

	for x := range w + 1 {
		for y := range h + 1 {
			_, _, _, ma := resizedBaskMask.At(x, y).RGBA()
			sr, sg, sb, _ := dst.At(x, y).RGBA() // dst has the proper color model

			newCol := color.RGBA{R: uint8(sr), G: uint8(sg), B: uint8(sb), A: 255 - uint8(ma)}
			dst.Set(x, y, newCol)
		}
	}

	return dst, nil
}

// AlreadyQuantated makes the quantize func return the included palette.
type AlreadyQuantated struct {
	palette color.Palette
}

func (ap AlreadyQuantated) Quantize(_ color.Palette, _ image.Image) color.Palette {
	return ap.palette
}

type QuantizerWithTransparencyGuarenteed struct {
	quantize.MedianCutQuantizer
}

// Quantize takes the normal palette returned by the normal quantizer and ensures it has a transparent
// color included as well.
func (q QuantizerWithTransparencyGuarenteed) Quantize(p color.Palette, m image.Image) color.Palette {
	palette := q.MedianCutQuantizer.Quantize(p, m)
	palette[0] = color.RGBA{0, 0, 0, 0} // force a transparent palette in there.

	return palette
}

// ImageToGif turns an image into a gif.
func (s *Server) ImageToGif(img image.Image) (io.Reader, error) {
	buff := bytes.Buffer{}
	if err := gif.Encode(&buff, img, &gif.Options{Quantizer: QuantizerWithTransparencyGuarenteed{quantize.MedianCutQuantizer{}}}); err != nil {
		return nil, nil
	}

	return &buff, nil
}
