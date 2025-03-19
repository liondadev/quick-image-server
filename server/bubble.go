package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/liondadev/quick-image-server/server/bubble"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/ericpauley/go-quantize/quantize"
)

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

	// Ensure we have a draw.Image, unlike the jpeg library.
	dst := normalizeImage(src)
	bubble.StdDrawer.Draw(dst, dst.Bounds(), dst, image.Point{})

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
	if err := gif.Encode(&buff, img, &gif.Options{Quantizer: QuantizerWithTransparencyGuarenteed{quantize.MedianCutQuantizer{}}, Drawer: bubble.Drawer{Base: draw.FloydSteinberg, Mask: bubble.Mask}}); err != nil {
		return nil, nil
	}

	return &buff, nil
}
