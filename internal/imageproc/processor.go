package imageproc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
)

const (
	// maxDecodePixels 限制单张图片解码后的像素总量，防止图片炸弹。
	maxDecodePixels = int64(100 * 1000 * 1000)
	// maxSourceBytes 限制单次读取的源图片大小。
	maxSourceBytes = int64(256 << 20)
	// cacheKeyVersion 缓存键版本，处理逻辑变更时递增以失效旧缓存。
	cacheKeyVersion = "v1"
)

// ErrImageTooLarge 表示图片像素或体积超限。
var ErrImageTooLarge = errors.New("imageproc: image too large")

// Options 描述一次图片处理请求。
type Options struct {
	Width     int     // 目标宽度，0 表示按高度自适应
	Height    int     // 目标高度，0 表示按宽度自适应
	Fit       string  // 缩放模式
	Quality   int     // 输出质量 1-100
	Format    string  // 期望输出格式，空表示沿用源格式
	Blur      float64 // 高斯模糊 sigma
	Gray      bool    // 是否灰度
	NoUpscale bool    // 是否禁止放大
	Watermark bool    // 是否叠加水印
}

// Key 返回选项的规范化描述，用于缓存键。
func (o Options) Key() string {
	return fmt.Sprintf("w%d_h%d_f%s_q%d_t%s_b%.2f_g%t_u%t_w%t",
		o.Width, o.Height, o.Fit, o.Quality, o.Format, o.Blur, o.Gray, o.NoUpscale, o.Watermark)
}

// Result 是图片处理结果。
type Result struct {
	Data   []byte
	Format string
	Mime   string
	Width  int
	Height int
	Cached bool
}

// Processor 组合解码、变换、编码与磁盘缓存。
type Processor struct {
	cfg   config.ImageConfig
	cache *Cache
	log   *logger.Logger

	wmOnce sync.Once
	wmImg  image.Image
	wmErr  error
}

// New 创建图片处理器，按配置初始化磁盘缓存。
func New(cfg config.ImageConfig, log *logger.Logger) *Processor {
	p := &Processor{cfg: cfg, log: log}
	if cfg.Dynamic.CacheEnabled {
		dir := cfg.Dynamic.CacheDir
		if dir == "" {
			dir = "./data/cache/imageproc"
		}
		c, err := NewCache(dir, cfg.Dynamic.CacheMaxSize, cfg.Dynamic.CacheTTL)
		if err != nil {
			if log != nil {
				log.Warn("init image cache failed", logger.Err(err))
			}
		} else {
			p.cache = c
		}
	}
	return p
}

// Cache 返回磁盘缓存，未启用时返回 nil。
func (p *Processor) Cache() *Cache { return p.cache }

// ThumbnailEnabled 返回是否启用缩略图。
func (p *Processor) ThumbnailEnabled() bool { return p.cfg.Thumbnail.Enabled }

// DynamicEnabled 返回是否启用动态处理。
func (p *Processor) DynamicEnabled() bool { return p.cfg.Dynamic.Enabled }

// ThumbnailSizes 返回配置的预设缩略图尺寸。
func (p *Processor) ThumbnailSizes() []config.ThumbSizeConfig { return p.cfg.Thumbnail.Sizes }

// ParseOptions 从查询参数解析动态处理选项，返回选项与是否需要处理。
func (p *Processor) ParseOptions(q url.Values) (Options, bool, error) {
	opt := Options{
		Fit:       FitInside,
		Quality:   p.cfg.Dynamic.DefaultQuality,
		NoUpscale: true,
	}
	if opt.Quality <= 0 {
		opt.Quality = 85
	}
	need := false

	if s := strings.TrimSpace(q.Get("w")); s != "" {
		w, err := strconv.Atoi(s)
		if err != nil || w <= 0 {
			return opt, false, errs.ErrInvalidParam.WithDetail("w 参数无效")
		}
		if maxW := p.cfg.Dynamic.MaxWidth; maxW > 0 && w > maxW {
			w = maxW
		}
		opt.Width = w
		need = true
	}
	if s := strings.TrimSpace(q.Get("h")); s != "" {
		h, err := strconv.Atoi(s)
		if err != nil || h <= 0 {
			return opt, false, errs.ErrInvalidParam.WithDetail("h 参数无效")
		}
		if maxH := p.cfg.Dynamic.MaxHeight; maxH > 0 && h > maxH {
			h = maxH
		}
		opt.Height = h
		need = true
	}
	if s := strings.TrimSpace(q.Get("fit")); s != "" {
		f := strings.ToLower(s)
		switch f {
		case FitCover, FitContain, FitFill, FitInside, FitOutside:
			opt.Fit = f
		default:
			return opt, false, errs.ErrInvalidParam.WithDetail("fit 参数无效")
		}
	}
	if s := strings.TrimSpace(q.Get("q")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > 100 {
			return opt, false, errs.ErrInvalidParam.WithDetail("q 参数需为 1-100")
		}
		opt.Quality = v
	}
	if s := strings.TrimSpace(q.Get("format")); s != "" {
		f := NormalizeFormat(s)
		if !p.formatAllowed(f) {
			return opt, false, errs.ErrInvalidParam.WithDetail("不支持的输出格式: " + s)
		}
		opt.Format = f
		need = true
	}
	if s := strings.TrimSpace(q.Get("blur")); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 || v > 100 {
			return opt, false, errs.ErrInvalidParam.WithDetail("blur 参数需为 0-100")
		}
		if v > 0 {
			opt.Blur = v / 10
			if opt.Blur < 0.3 {
				opt.Blur = 0.3
			}
			if opt.Blur > 20 {
				opt.Blur = 20
			}
			need = true
		}
	}
	if s := strings.TrimSpace(q.Get("gray")); s != "" {
		opt.Gray = s == "1" || strings.EqualFold(s, "true") || strings.EqualFold(s, "yes")
		if opt.Gray {
			need = true
		}
	}
	opt.Watermark = p.cfg.Watermark.Enabled
	return opt, need, nil
}

func (p *Processor) formatAllowed(format string) bool {
	format = NormalizeFormat(format)
	if len(p.cfg.Dynamic.AllowedFormats) == 0 {
		return CanEncode(format)
	}
	for _, f := range p.cfg.Dynamic.AllowedFormats {
		if NormalizeFormat(f) == format {
			return true
		}
	}
	return false
}

// ThumbOption 根据尺寸名或像素宽度构造缩略图选项。
func (p *Processor) ThumbOption(name string) (Options, bool) {
	if !p.cfg.Thumbnail.Enabled {
		return Options{}, false
	}
	quality := p.cfg.Thumbnail.Quality
	if quality <= 0 {
		quality = 82
	}
	base := Options{
		Fit:       FitInside,
		Quality:   quality,
		Format:    p.cfg.Thumbnail.Format,
		NoUpscale: true,
	}
	if w, err := strconv.Atoi(name); err == nil && w > 0 {
		if maxW := p.cfg.Dynamic.MaxWidth; maxW > 0 && w > maxW {
			w = maxW
		}
		base.Width = w
		return base, true
	}
	for _, size := range p.cfg.Thumbnail.Sizes {
		if strings.EqualFold(size.Name, name) {
			base.Width = size.Width
			return base, true
		}
	}
	return Options{}, false
}

// outputFormat 决定实际输出格式：优先请求格式，其次源格式，最后降级 jpeg。
func outputFormat(srcFormat, requested string) string {
	req := NormalizeFormat(requested)
	if req != "" {
		if CanEncode(req) {
			return req
		}
		return FormatJPEG
	}
	if CanEncode(srcFormat) {
		return NormalizeFormat(srcFormat)
	}
	return FormatJPEG
}

// Render 解码并处理图片，返回编码后的结果。
func (p *Processor) Render(src []byte, opt Options) (*Result, error) {
	if len(src) == 0 {
		return nil, ErrUnsupportedFormat
	}
	info, err := Probe(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	if info.Width > 0 && info.Height > 0 && int64(info.Width)*int64(info.Height) > maxDecodePixels {
		return nil, ErrImageTooLarge
	}
	if info.Format == FormatSVG {
		return nil, ErrUnsupportedFormat
	}
	img, srcFormat, err := Decode(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	img = p.Transform(img, opt)
	format := outputFormat(srcFormat, opt.Format)
	data, err := Encode(img, format, opt.Quality)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	return &Result{
		Data:   data,
		Format: format,
		Mime:   MimeOf(format),
		Width:  b.Dx(),
		Height: b.Dy(),
	}, nil
}

// RenderCached 先查缓存，未命中时通过 load 读取源图并处理，结果写回缓存。
// sourceID 是源图稳定标识（如图片 hash），用于构造缓存键。
func (p *Processor) RenderCached(ctx context.Context, sourceID string, load func(context.Context) (io.ReadCloser, error), opt Options) (*Result, error) {
	if p.cache == nil {
		return p.renderFromLoader(ctx, load, opt)
	}
	key := p.cacheKey(sourceID, opt)
	if data, format, ok := p.cache.GetAny(key); ok {
		return &Result{Data: data, Format: format, Mime: MimeOf(format), Cached: true}, nil
	}
	res, err := p.renderFromLoader(ctx, load, opt)
	if err != nil {
		return nil, err
	}
	if err := p.cache.Put(key, res.Format, res.Data); err != nil && p.log != nil {
		p.log.Warn("write image cache failed", logger.Err(err))
	}
	return res, nil
}

func (p *Processor) renderFromLoader(ctx context.Context, load func(context.Context) (io.ReadCloser, error), opt Options) (*Result, error) {
	rc, err := load(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSourceBytes {
		return nil, ErrImageTooLarge
	}
	return p.Render(data, opt)
}

func (p *Processor) cacheKey(sourceID string, opt Options) string {
	h := sha256.New()
	h.Write([]byte(cacheKeyVersion))
	h.Write([]byte("|"))
	h.Write([]byte(sourceID))
	h.Write([]byte("|"))
	h.Write([]byte(opt.Key()))
	return hex.EncodeToString(h.Sum(nil))
}
