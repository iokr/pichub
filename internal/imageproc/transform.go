package imageproc

import (
	"image"
	"math"

	"github.com/disintegration/imaging"
)

// 缩放模式。
const (
	FitCover   = "cover"   // 等比缩放并裁剪，填满目标区域
	FitContain = "contain" // 等比缩放，完整放入目标区域
	FitFill    = "fill"    // 拉伸到目标宽高，不保持比例
	FitInside  = "inside"  // 等比缩放到目标区域，不放大
	FitOutside = "outside" // 等比缩放到覆盖目标区域，不裁剪
)

// Resize 按选项缩放图片。
func Resize(img image.Image, opt Options) image.Image {
	w, h := opt.Width, opt.Height
	if w <= 0 && h <= 0 {
		return img
	}
	b := img.Bounds()
	ow, oh := b.Dx(), b.Dy()
	if ow <= 0 || oh <= 0 {
		return img
	}
	if opt.NoUpscale {
		if w > ow {
			w = ow
		}
		if h > oh {
			h = oh
		}
	}
	switch opt.Fit {
	case FitCover:
		if w <= 0 || h <= 0 {
			return resizeByOneSide(img, w, h)
		}
		return imaging.Fill(img, w, h, imaging.Center, imaging.Lanczos)
	case FitContain:
		if w <= 0 || h <= 0 {
			return resizeByOneSide(img, w, h)
		}
		return imaging.Fit(img, w, h, imaging.Lanczos)
	case FitFill:
		if w <= 0 {
			w = ow
		}
		if h <= 0 {
			h = oh
		}
		return imaging.Resize(img, w, h, imaging.Lanczos)
	case FitOutside:
		return resizeOutside(img, w, h)
	default: // inside
		return resizeByOneSide(img, w, h)
	}
}

func resizeByOneSide(img image.Image, w, h int) image.Image {
	if w > 0 {
		return imaging.Resize(img, w, 0, imaging.Lanczos)
	}
	if h > 0 {
		return imaging.Resize(img, 0, h, imaging.Lanczos)
	}
	return img
}

func resizeOutside(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	ow, oh := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return resizeByOneSide(img, w, h)
	}
	scale := math.Max(float64(w)/float64(ow), float64(h)/float64(oh))
	nw := int(math.Round(float64(ow) * scale))
	nh := int(math.Round(float64(oh) * scale))
	return imaging.Resize(img, nw, nh, imaging.Lanczos)
}

// Transform 执行完整变换链：缩放 → 模糊 → 灰度 → 水印。
func (p *Processor) Transform(img image.Image, opt Options) image.Image {
	img = Resize(img, opt)
	if opt.Blur > 0 {
		img = imaging.Blur(img, opt.Blur)
	}
	if opt.Gray {
		img = imaging.Grayscale(img)
	}
	if opt.Watermark {
		img = p.applyWatermark(img)
	}
	return img
}
