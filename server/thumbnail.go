package server

import (
	"bytes"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

const (
	ThumbnailWidth  uint = 1920 / 4
	ThumbnailHeight uint = 1080 / 4
)

var AllowedThumbnailMimeTypes = []string{"image/jpeg", "image/png"}

// MakeThumbnail creates a thumbnail (png) image from an original upload.
func (s *Server) MakeThumbnail(mime string, original io.Reader) (io.Reader, error) {
	var img image.Image

	switch mime {
	case "image/jpeg":
		dec, err := jpeg.Decode(original)
		if err != nil {
			return nil, fmt.Errorf("decode jpeg: %w", err)
		}
		img = dec
	case "image/png":
		dec, err := png.Decode(original)
		if err != nil {
			return nil, fmt.Errorf("decode png: %w", err)
		}
		img = dec
	default:
		return nil, fmt.Errorf("mime type '%s' can't be used to create thumbnails", mime)
	}

	thumbImg := resize.Resize(ThumbnailWidth, ThumbnailHeight, img, resize.Lanczos3)
	buff := new(bytes.Buffer)
	if err := png.Encode(buff, thumbImg); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return buff, nil
}
