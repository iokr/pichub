package imageproc

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"io"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

// ErrUnsupportedFormat 表示格式不支持解码或处理。
var ErrUnsupportedFormat = errors.New("imageproc: unsupported format")

// Info 是图片基本信息。
type Info struct {
	Format string
	Width  int
	Height int
}

// Decode 解码图片并返回格式，JPEG/TIFF 自动应用 EXIF 方向。
func Decode(r io.Reader) (image.Image, string, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	head, err := br.Peek(512)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, "", err
	}
	format := DetectFormat(head)
	switch format {
	case FormatJPEG, FormatPNG, FormatGIF, FormatBMP, FormatTIFF:
		img, derr := imaging.Decode(br, imaging.AutoOrientation(true))
		if derr != nil {
			return nil, format, fmt.Errorf("decode %s: %w", format, derr)
		}
		return img, format, nil
	case FormatWebP:
		img, derr := webp.Decode(br, webp.Options{AutoRotate: true})
		if derr != nil {
			return nil, format, fmt.Errorf("decode webp: %w", derr)
		}
		return img, format, nil
	default:
		return nil, format, ErrUnsupportedFormat
	}
}

// Probe 读取图片尺寸与格式，用于上传校验。
func Probe(r io.Reader) (*Info, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	head, err := br.Peek(512)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, err
	}
	format := DetectFormat(head)
	info := &Info{Format: format}
	switch format {
	case FormatSVG:
		// SVG 为矢量格式，不计算像素尺寸。
		return info, nil
	case FormatWebP:
		cfg, derr := webp.DecodeConfig(br)
		if derr != nil {
			return nil, fmt.Errorf("probe webp: %w", derr)
		}
		info.Width, info.Height = cfg.Width, cfg.Height
		return info, nil
	case FormatJPEG, FormatPNG, FormatGIF, FormatBMP, FormatTIFF:
		cfg, _, derr := image.DecodeConfig(br)
		if derr != nil {
			return nil, fmt.Errorf("probe %s: %w", format, derr)
		}
		info.Width, info.Height = cfg.Width, cfg.Height
		return info, nil
	default:
		return nil, ErrUnsupportedFormat
	}
}
