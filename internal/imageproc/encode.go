package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/gen2brain/webp"
)

// Encode 把图片编码为指定格式，quality 仅对 jpeg/webp 有损编码生效。
func Encode(img image.Image, format string, quality int) ([]byte, error) {
	format = NormalizeFormat(format)
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	buf := &bytes.Buffer{}
	switch format {
	case FormatJPEG:
		// JPEG 不支持透明通道，统一转换为 RGB 后编码。
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode jpeg: %w", err)
		}
	case FormatPNG:
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := enc.Encode(buf, img); err != nil {
			return nil, fmt.Errorf("encode png: %w", err)
		}
	case FormatGIF:
		if err := gif.Encode(buf, img, nil); err != nil {
			return nil, fmt.Errorf("encode gif: %w", err)
		}
	case FormatWebP:
		if err := webp.Encode(buf, img, webp.Options{Quality: quality, Method: webp.DefaultMethod}); err != nil {
			return nil, fmt.Errorf("encode webp: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	return buf.Bytes(), nil
}
