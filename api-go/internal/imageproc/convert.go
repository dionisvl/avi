package imageproc

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/gen2brain/heic"
	"github.com/gen2brain/webp"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	xwebp "golang.org/x/image/webp"
)

const (
	DefaultQuality   = 75
	DefaultMaxSide   = 2048
	DefaultMaxPixels = 40_000_000 // 40 MP
)

var (
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrTooLarge          = errors.New("image exceeds pixel budget")
	ErrDecode            = errors.New("decode failed")
)

type Result struct {
	Data       []byte
	Width      int
	Height     int
	SourceMIME string
}

type Options struct {
	Quality   int
	MaxSide   int
	MaxPixels int
}

type Converter struct {
	opts Options
}

func New(opts Options) *Converter {
	if opts.Quality == 0 {
		opts.Quality = DefaultQuality
	}
	if opts.MaxSide == 0 {
		opts.MaxSide = DefaultMaxSide
	}
	if opts.MaxPixels == 0 {
		opts.MaxPixels = DefaultMaxPixels
	}
	return &Converter{opts: opts}
}

func (c *Converter) Convert(src []byte) (Result, error) {
	mime := DetectMIME(src)
	if !IsSupported(mime) {
		return Result{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, mime)
	}

	if err := c.guardPixels(src, mime); err != nil {
		return Result{}, err
	}

	img, err := decode(src, mime)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrDecode, err)
	}

	img = resizeMaxSide(img, c.opts.MaxSide)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: c.opts.Quality}); err != nil {
		return Result{}, fmt.Errorf("encode webp: %w", err)
	}

	b := img.Bounds()
	return Result{
		Data:       buf.Bytes(),
		Width:      b.Dx(),
		Height:     b.Dy(),
		SourceMIME: mime,
	}, nil
}

func DetectMIME(src []byte) string {
	if len(src) < 12 {
		return "application/octet-stream"
	}
	// HEIC/HEIF: ftyp box
	if len(src) >= 12 && (string(src[4:8]) == "ftyp") {
		brand := string(src[8:12])
		switch brand {
		case "heic", "heix", "hevc", "hevx", "mif1", "msf1":
			return "image/heic"
		}
	}
	// WebP: RIFF....WEBP
	if len(src) >= 12 && string(src[0:4]) == "RIFF" && string(src[8:12]) == "WEBP" {
		return "image/webp"
	}
	// BMP
	if len(src) >= 2 && src[0] == 0x42 && src[1] == 0x4D {
		return "image/bmp"
	}
	// delegate rest to stdlib sniffer (handles jpeg, png, gif)
	return detectViaStdlib(src)
}

func detectViaStdlib(src []byte) string {
	switch {
	case len(src) >= 3 && src[0] == 0xFF && src[1] == 0xD8 && src[2] == 0xFF:
		return "image/jpeg"
	case len(src) >= 8 && string(src[0:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(src) >= 6 && (string(src[0:6]) == "GIF87a" || string(src[0:6]) == "GIF89a"):
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func IsSupported(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp", "image/heic":
		return true
	default:
		return false
	}
}

func (c *Converter) guardPixels(src []byte, mime string) error {
	var cfg image.Config
	var err error
	r := bytes.NewReader(src)
	switch mime {
	case "image/heic":
		cfg, err = heic.DecodeConfig(r)
	case "image/webp":
		cfg, err = xwebp.DecodeConfig(r)
	case "image/bmp":
		cfg, err = bmp.DecodeConfig(r)
	default:
		var format string
		cfg, format, err = image.DecodeConfig(r)
		_ = format
	}
	if err != nil {
		return fmt.Errorf("%w: read config: %w", ErrDecode, err)
	}
	if cfg.Width*cfg.Height > c.opts.MaxPixels {
		return ErrTooLarge
	}
	return nil
}

func decode(src []byte, mime string) (image.Image, error) {
	r := bytes.NewReader(src)
	switch mime {
	case "image/heic":
		return heic.Decode(r)
	case "image/webp":
		return xwebp.Decode(r)
	case "image/bmp":
		return bmp.Decode(r)
	case "image/gif":
		g, err := gif.DecodeAll(r)
		if err != nil {
			return nil, err
		}
		return g.Image[0], nil
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	default:
		img, _, err := image.Decode(r)
		return img, err
	}
}

func resizeMaxSide(img image.Image, maxSide int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return img
	}
	var nw, nh int
	if w >= h {
		nw = maxSide
		nh = h * maxSide / w
	} else {
		nh = maxSide
		nw = w * maxSide / h
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

// ConvertReader is a convenience wrapper for io.Reader input.
func (c *Converter) ConvertReader(r io.Reader) (Result, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return Result{}, fmt.Errorf("read: %w", err)
	}
	return c.Convert(src)
}
