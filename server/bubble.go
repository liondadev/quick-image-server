package server

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/ericpauley/go-quantize/quantize"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
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

	dst, ok := src.(draw.Image)
	if !ok {
		return nil, errors.New("draw is not assertable to a draw.image")
	}

	mb := (*bubbleMaskImg).Bounds()
	mh := mb.Dy()

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// Create the mask image
	mask := image.NewRGBA(image.Rect(0, 0, w, h))
	resizedBaskMask := resize.Resize(uint(w), uint(mh), *bubbleMaskImg, resize.Lanczos3)
	draw.Draw(mask, image.Rect(0, 0, w, mh), resizedBaskMask, image.Point{}, draw.Over)

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			_, _, _, ma := resizedBaskMask.At(x, y).RGBA()
			sr, sg, sb, _ := src.At(x, y).RGBA()
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

// ImageToGif turns an image into a gif.
func (s *Server) ImageToGif(img image.Image) (io.Reader, error) {
	buff := bytes.Buffer{}
	if err := gif.Encode(&buff, img, &gif.Options{Quantizer: quantize.MedianCutQuantizer{}}); err != nil {
		return nil, nil
	}

	return &buff, nil
}
