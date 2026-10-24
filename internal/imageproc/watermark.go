package imageproc

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// 水印位置。
const (
	PositionTopLeft     = "top-left"
	PositionTopRight    = "top-right"
	PositionBottomLeft  = "bottom-left"
	PositionBottomRight = "bottom-right"
	PositionCenter      = "center"
)

const watermarkMargin = 12

// applyWatermark 按配置给图片叠加水印，失败时原样返回。
func (p *Processor) applyWatermark(base image.Image) image.Image {
	wm := p.cfg.Watermark
	if !wm.Enabled {
		return base
	}
	if strings.EqualFold(wm.Type, "image") && wm.ImagePath != "" {
		if out, err := p.imageWatermark(base); err == nil {
			return out
		} else if p.log != nil {
			p.log.Warn("apply image watermark failed: " + err.Error())
		}
		return base
	}
	if wm.Text == "" {
		return base
	}
	return textWatermark(base, wm.Text, wm.FontSize, wm.Color, wm.Opacity, wm.Position)
}

// textWatermark 使用内置 7x13 位图字体绘制文本水印（仅支持 ASCII）。
func textWatermark(base image.Image, text string, fontSize float64, colorHex string, opacity float64, position string) image.Image {
	face := basicfont.Face7x13
	n := len([]rune(text))
	if n == 0 {
		return base
	}
	scale := 1
	if fontSize > float64(face.Height) {
		scale = int(math.Round(fontSize / float64(face.Height)))
	}
	if scale < 1 {
		scale = 1
	}
	textW := face.Advance * n
	textH := face.Height
	tmp := image.NewRGBA(image.Rect(0, 0, textW, textH))
	col := parseHexColor(colorHex)
	drawer := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(0, face.Ascent),
	}
	drawer.DrawString(text)

	var layer image.Image = tmp
	if scale > 1 {
		layer = imaging.Resize(tmp, textW*scale, textH*scale, imaging.NearestNeighbor)
	}
	dst := toRGBA(base)
	size := image.Pt(layer.Bounds().Dx(), layer.Bounds().Dy())
	rect := positionRect(dst.Bounds(), size, watermarkMargin, position)
	mask := image.NewUniform(color.Alpha{A: uint8(clampOpacity(opacity) * 255)})
	draw.DrawMask(dst, rect, layer, image.Point{}, mask, image.Point{}, draw.Over)
	return dst
}

// imageWatermark 叠加图片水印，水印宽度不超过底图的四分之一。
func (p *Processor) imageWatermark(base image.Image) (image.Image, error) {
	img, err := p.loadWatermarkImage()
	if err != nil {
		return nil, err
	}
	b := base.Bounds()
	maxW := b.Dx() / 4
	if maxW < 32 {
		maxW = b.Dx() / 2
	}
	wm := img
	if wm.Bounds().Dx() > maxW {
		wm = imaging.Resize(wm, maxW, 0, imaging.Lanczos)
	}
	dst := toRGBA(base)
	layer := toRGBA(wm)
	rect := positionRect(dst.Bounds(), layer.Bounds().Size(), watermarkMargin, p.cfg.Watermark.Position)
	mask := image.NewUniform(color.Alpha{A: uint8(clampOpacity(p.cfg.Watermark.Opacity) * 255)})
	draw.DrawMask(dst, rect, layer, image.Point{}, mask, image.Point{}, draw.Over)
	return dst, nil
}

// loadWatermarkImage 延迟加载并缓存图片水印文件。
func (p *Processor) loadWatermarkImage() (image.Image, error) {
	p.wmOnce.Do(func() {
		path := p.cfg.Watermark.ImagePath
		if path == "" {
			return
		}
		f, err := os.Open(path)
		if err != nil {
			p.wmErr = err
			return
		}
		defer f.Close()
		img, _, err := Decode(f)
		if err != nil {
			p.wmErr = err
			return
		}
		p.wmImg = img
	})
	return p.wmImg, p.wmErr
}

// toRGBA 把任意图片转换为原点对齐的 RGBA 图片，必要时复制一份。
func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok && r.Bounds().Min == (image.Point{}) {
		return r
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

// positionRect 计算水印在底图中的位置。
func positionRect(bounds image.Rectangle, size image.Point, margin int, position string) image.Rectangle {
	w, h := size.X, size.Y
	var x, y int
	switch strings.ToLower(position) {
	case PositionTopLeft:
		x, y = margin, margin
	case PositionTopRight:
		x, y = bounds.Dx()-w-margin, margin
	case PositionBottomLeft:
		x, y = margin, bounds.Dy()-h-margin
	case PositionCenter:
		x, y = (bounds.Dx()-w)/2, (bounds.Dy()-h)/2
	default: // bottom-right
		x, y = bounds.Dx()-w-margin, bounds.Dy()-h-margin
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > bounds.Dx()-w {
		x = bounds.Dx() - w
	}
	if y > bounds.Dy()-h {
		y = bounds.Dy() - h
	}
	return image.Rect(x, y, x+w, y+h)
}

// parseHexColor 解析 #RGB / #RRGGBB / #RRGGBBAA 颜色，失败时返回白色。
func parseHexColor(s string) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	parse := func(part string) uint8 {
		v, err := strconv.ParseUint(part, 16, 8)
		if err != nil {
			return 255
		}
		return uint8(v)
	}
	switch len(s) {
	case 3:
		return color.RGBA{parse(s[0:1] + s[0:1]), parse(s[1:2] + s[1:2]), parse(s[2:3] + s[2:3]), 255}
	case 6:
		return color.RGBA{parse(s[0:2]), parse(s[2:4]), parse(s[4:6]), 255}
	case 8:
		return color.RGBA{parse(s[0:2]), parse(s[2:4]), parse(s[4:6]), parse(s[6:8])}
	default:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
}

func clampOpacity(v float64) float64 {
	if v <= 0 {
		return 1
	}
	if v > 1 {
		return 1
	}
	return v
}
