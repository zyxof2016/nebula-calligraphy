package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	defaultGlyphRenderSize = 512
	defaultRenderFontFile  = "MaShanZheng-Regular.ttf"
)

type GlyphRenderOptions struct {
	Grid string
	Size int
}

type GlyphImageRenderer struct {
	fontPath string
	cacheDir string
}

func NewGlyphImageRenderer(fontPath, cacheDir string) *GlyphImageRenderer {
	return &GlyphImageRenderer{
		fontPath: strings.TrimSpace(fontPath),
		cacheDir: strings.TrimSpace(cacheDir),
	}
}

func (r *GlyphImageRenderer) RenderPNG(glyph model.Glyph, opts GlyphRenderOptions) ([]byte, error) {
	if strings.TrimSpace(glyph.GlyphID) == "" || strings.TrimSpace(glyph.Character) == "" {
		return nil, errors.New("glyph id and character are required")
	}
	if opts.Size <= 0 {
		opts.Size = defaultGlyphRenderSize
	}
	if opts.Grid == "" {
		opts.Grid = "none"
	}
	cachePath := r.cachePath(glyph, opts)
	if cachePath != "" {
		if content, err := os.ReadFile(cachePath); err == nil {
			return content, nil
		}
	}

	fontPath := r.fontPath
	if fontPath == "" {
		fontPath = FindDefaultRenderFontPath()
	}
	if fontPath == "" {
		return nil, errors.New("server render font is not configured")
	}
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}
	parsedFont, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    float64(opts.Size) * 0.70,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, opts.Size, opts.Size))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 255, G: 251, B: 242, A: 255}), image.Point{}, draw.Src)
	drawGuides(img, opts.Grid)

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 30, G: 27, B: 22, A: 255}),
		Face: face,
	}
	advance := drawer.MeasureString(glyph.Character).Ceil()
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	drawer.Dot = fixed.P(
		(opts.Size-advance)/2,
		(opts.Size+ascent-descent)/2,
	)
	drawer.DrawString(glyph.Character)

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	content := out.Bytes()
	if cachePath != "" {
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
			_ = os.WriteFile(cachePath, content, 0o644)
		}
	}
	return content, nil
}

func (r *GlyphImageRenderer) cachePath(glyph model.Glyph, opts GlyphRenderOptions) string {
	if r == nil || r.cacheDir == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(glyph.GlyphID + "\x00" + glyph.Character + "\x00" + opts.Grid + "\x00" + strconv.Itoa(opts.Size)))
	return filepath.Join(r.cacheDir, hex.EncodeToString(hash[:])+".png")
}

func drawGuides(img *image.RGBA, grid string) {
	bounds := img.Bounds()
	size := bounds.Dx()
	border := color.RGBA{R: 154, G: 106, B: 58, A: 210}
	guide := color.RGBA{R: 154, G: 106, B: 58, A: 95}
	drawLine(img, 0, 0, size-1, 0, border)
	drawLine(img, 0, size-1, size-1, size-1, border)
	drawLine(img, 0, 0, 0, size-1, border)
	drawLine(img, size-1, 0, size-1, size-1, border)
	switch grid {
	case "mi":
		drawLine(img, size/2, 0, size/2, size-1, guide)
		drawLine(img, 0, size/2, size-1, size/2, guide)
		drawLine(img, 0, 0, size-1, size-1, guide)
		drawLine(img, size-1, 0, 0, size-1, guide)
	case "jiugong":
		for i := 1; i < 3; i++ {
			p := size * i / 3
			drawLine(img, p, 0, p, size-1, guide)
			drawLine(img, 0, p, size-1, p, guide)
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		img.SetRGBA(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func FindDefaultRenderFontPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, candidate := range []string{
			filepath.Join(cwd, "assets", "fonts", defaultRenderFontFile),
			filepath.Join(cwd, "..", "..", "assets", "fonts", defaultRenderFontFile),
			filepath.Join(cwd, "..", "..", "apps", "mobile", "assets", "fonts", defaultRenderFontFile),
		} {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return filepath.Clean(candidate)
			}
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return ""
		}
		cwd = parent
	}
}
