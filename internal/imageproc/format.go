// Package imageproc 提供图片解码、缩放、格式转换、水印与磁盘缓存能力。
package imageproc

import (
	"bytes"
	"strings"
)

// 支持的图片格式。
const (
	FormatJPEG = "jpeg"
	FormatPNG  = "png"
	FormatGIF  = "gif"
	FormatWebP = "webp"
	FormatBMP  = "bmp"
	FormatTIFF = "tiff"
	FormatAVIF = "avif"
	FormatSVG  = "svg"
)

// 可解码格式，用于校验上传文件。
var decodableFormats = map[string]bool{
	FormatJPEG: true,
	FormatPNG:  true,
	FormatGIF:  true,
	FormatWebP: true,
	FormatBMP:  true,
	FormatTIFF: true,
}

var mimeOf = map[string]string{
	FormatJPEG: "image/jpeg",
	FormatPNG:  "image/png",
	FormatGIF:  "image/gif",
	FormatWebP: "image/webp",
	FormatBMP:  "image/bmp",
	FormatTIFF: "image/tiff",
	FormatAVIF: "image/avif",
	FormatSVG:  "image/svg+xml",
}

var extOf = map[string]string{
	FormatJPEG: "jpg",
	FormatPNG:  "png",
	FormatGIF:  "gif",
	FormatWebP: "webp",
	FormatBMP:  "bmp",
	FormatTIFF: "tiff",
	FormatAVIF: "avif",
	FormatSVG:  "svg",
}

var formatOfExt = map[string]string{
	"jpg":  FormatJPEG,
	"jpeg": FormatJPEG,
	"png":  FormatPNG,
	"gif":  FormatGIF,
	"webp": FormatWebP,
	"bmp":  FormatBMP,
	"tif":  FormatTIFF,
	"tiff": FormatTIFF,
	"avif": FormatAVIF,
	"svg":  FormatSVG,
}

// NormalizeFormat 归一化格式名：小写、去点号、jpg 归一为 jpeg。
func NormalizeFormat(s string) string {
	f := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(s, ".")))
	if f == "jpg" {
		return FormatJPEG
	}
	return f
}

// MimeOf 返回格式对应的 MIME 类型。
func MimeOf(format string) string {
	if m, ok := mimeOf[NormalizeFormat(format)]; ok {
		return m
	}
	return "application/octet-stream"
}

// ExtOf 返回格式对应的文件扩展名。
func ExtOf(format string) string {
	if e, ok := extOf[NormalizeFormat(format)]; ok {
		return e
	}
	return "bin"
}

// FormatOfExt 根据扩展名返回图片格式。
func FormatOfExt(ext string) string {
	return formatOfExt[strings.ToLower(strings.TrimPrefix(ext, "."))]
}

// CanEncode 判断是否支持编码该格式。
func CanEncode(format string) bool {
	switch NormalizeFormat(format) {
	case FormatJPEG, FormatPNG, FormatGIF, FormatWebP:
		return true
	}
	return false
}

// CanDecode 判断是否支持解码该格式。
func CanDecode(format string) bool {
	return decodableFormats[NormalizeFormat(format)]
}

// EncodedFormats 返回全部可编码格式。
func EncodedFormats() []string {
	return []string{FormatJPEG, FormatPNG, FormatGIF, FormatWebP}
}

// DetectFormat 通过文件头魔数识别图片格式，无法识别时返回空串。
func DetectFormat(data []byte) string {
	if len(data) < 12 {
		return ""
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return FormatJPEG
	case bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return FormatPNG
	case bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a")):
		return FormatGIF
	case bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return FormatWebP
	case bytes.HasPrefix(data, []byte("BM")):
		return FormatBMP
	case bytes.HasPrefix(data, []byte("II*\x00")) || bytes.HasPrefix(data, []byte("MM\x00*")):
		return FormatTIFF
	case bytes.Equal(data[4:8], []byte("ftyp")):
		brand := string(data[8:12])
		if brand == "avif" || brand == "avis" {
			return FormatAVIF
		}
	case looksLikeSVG(data):
		return FormatSVG
	}
	return ""
}

func looksLikeSVG(data []byte) bool {
	head := bytes.TrimLeft(data, " \t\r\n")
	if !bytes.HasPrefix(head, []byte("<")) {
		return false
	}
	lower := bytes.ToLower(head)
	if bytes.HasPrefix(lower, []byte("<?xml")) || bytes.HasPrefix(lower, []byte("<!doctype")) {
		return bytes.Contains(lower, []byte("<svg"))
	}
	return bytes.HasPrefix(lower, []byte("<svg"))
}

// IsImageExtension 判断扩展名是否属于常见图片类型。
func IsImageExtension(ext string) bool {
	return FormatOfExt(ext) != ""
}
