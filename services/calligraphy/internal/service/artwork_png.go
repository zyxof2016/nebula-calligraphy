package service

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const maxArtworkPNGSide = 1800

type ArtworkPNGRenderer struct {
	fontPath string
}

func NewArtworkPNGRenderer(fontPath string) *ArtworkPNGRenderer {
	return &ArtworkPNGRenderer{fontPath: strings.TrimSpace(fontPath)}
}

func (r *ArtworkPNGRenderer) Render(layout model.LayoutResult) ([]byte, error) {
	if layout.Paper.WidthCM <= 0 || layout.Paper.HeightCM <= 0 {
		return nil, errors.New("paper width_cm and height_cm must be positive")
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

	scale := math.Min(maxArtworkPNGSide/layout.Paper.WidthCM, maxArtworkPNGSide/layout.Paper.HeightCM)
	width := int(math.Round(layout.Paper.WidthCM * scale))
	height := int(math.Round(layout.Paper.HeightCM * scale))
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 251, G: 247, B: 239, A: 255}), image.Point{}, draw.Src)
	drawPaperTexture(img)
	drawRect(img, image.Rect(1, 1, width-2, height-2), color.RGBA{R: 139, G: 122, B: 90, A: 210})
	margin := int(math.Round(layout.MarginCM * scale))
	if margin > 0 && margin*2 < width && margin*2 < height {
		drawRect(img, image.Rect(margin, margin, width-margin, height-margin), color.RGBA{R: 139, G: 122, B: 90, A: 80})
	}

	for _, slot := range layout.Slots {
		size := slot.SizeCM * scale
		rhythm := 0.94 + float64(slot.Index%4)*0.025
		x := slot.XCM*scale + rhythmicOffset(slot.Index, size, true)
		y := slot.YCM*scale + rhythmicOffset(slot.Row, size, false)
		if err := drawCenteredString(img, parsedFont, slot.Character, x, y, size*rhythm, color.RGBA{R: 29, G: 27, B: 22, A: 255}); err != nil {
			return nil, err
		}
	}
	for _, slot := range layout.SignatureSlots {
		if strings.TrimSpace(slot.Text) == "" {
			continue
		}
		if err := drawCenteredString(img, parsedFont, slot.Text, slot.XCM*scale, slot.YCM*scale, math.Max(slot.SizeCM*scale, 12), color.RGBA{R: 61, G: 52, B: 40, A: 255}); err != nil {
			return nil, err
		}
	}
	for _, slot := range layout.SealSlots {
		drawSeal(img, slot, scale, parsedFont)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func drawPaperTexture(img *image.RGBA) {
	bounds := img.Bounds()
	for i := 1; i < 10; i++ {
		y := bounds.Min.Y + bounds.Dy()*i/10
		drawLine(img, bounds.Min.X+bounds.Dx()/18, y, bounds.Max.X-bounds.Dx()/18, y, color.RGBA{R: 169, G: 134, B: 82, A: 18})
	}
}

func drawRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	drawLine(img, rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y, c)
	drawLine(img, rect.Min.X, rect.Max.Y, rect.Max.X, rect.Max.Y, c)
	drawLine(img, rect.Min.X, rect.Min.Y, rect.Min.X, rect.Max.Y, c)
	drawLine(img, rect.Max.X, rect.Min.Y, rect.Max.X, rect.Max.Y, c)
}

func rhythmicOffset(index int, size float64, horizontal bool) float64 {
	if horizontal {
		if index%2 == 0 {
			return -size * 0.025
		}
		return size * 0.018
	}
	if index%2 == 0 {
		return -size * 0.012
	}
	return size * 0.018
}

func drawCenteredString(img *image.RGBA, parsedFont *opentype.Font, text string, centerX, centerY, size float64, ink color.RGBA) error {
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(ink),
		Face: face,
	}
	advance := drawer.MeasureString(text).Ceil()
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	drawer.Dot = fixed.P(
		int(math.Round(centerX))-advance/2,
		int(math.Round(centerY))+(ascent-descent)/2,
	)
	drawer.DrawString(text)
	return nil
}

func drawSeal(img *image.RGBA, slot model.TextSlot, scale float64, parsedFont *opentype.Font) {
	side := int(math.Round(math.Max(slot.SizeCM*scale, 24)))
	centerX := int(math.Round(slot.XCM * scale))
	centerY := int(math.Round(slot.YCM * scale))
	rect := image.Rect(centerX-side/2, centerY-side/2, centerX+side/2, centerY+side/2)
	fill := image.NewUniform(color.RGBA{R: 179, G: 38, B: 30, A: 34})
	draw.Draw(img, rect, fill, image.Point{}, draw.Over)
	drawRect(img, rect, color.RGBA{R: 179, G: 38, B: 30, A: 230})
	_ = drawCenteredString(img, parsedFont, "印", float64(centerX), float64(centerY), float64(side)*0.48, color.RGBA{R: 179, G: 38, B: 30, A: 255})
}
