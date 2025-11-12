package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	assets "github.com/ponyo877/suika-shaker/assets/image"
)

const (
	ScreenWidth  = 480
	ScreenHeight = 800
)

type ButtonConfig struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

var (
	SpeakerButtonConfig = ButtonConfig{X: ScreenWidth - 60, Y: 10, Width: 50, Height: 50}
)

type DialogConfig struct {
	Width        float32
	Height       float32
	X            float32
	Y            float32
	BorderWidth  float32
	Radius       float32
	ButtonY      float32
	RetryX       float32
	RetryWidth   float32
	RetryHeight  float32
	RetryRadius  float32
	XButtonSize  float32
	XButtonX     float32
	XButtonCenterX float32
	XButtonCenterY float32
}

func NewDialogConfig() DialogConfig {
	const (
		dialogWidth  = 340
		dialogHeight = 440
		dialogX      = (ScreenWidth - dialogWidth) / 2
		dialogY      = (ScreenHeight - dialogHeight) / 2
		borderWidth  = 10
		radius       = 25
		buttonY      = dialogY + dialogHeight - 85
		retryWidth   = 230
		retryHeight  = 50
		retryRadius  = 15
		retryX       = dialogX + 25
		xButtonSize  = 60
		xButtonX     = retryX + retryWidth + 15
	)

	return DialogConfig{
		Width:          dialogWidth,
		Height:         dialogHeight,
		X:              dialogX,
		Y:              dialogY,
		BorderWidth:    borderWidth,
		Radius:         radius,
		ButtonY:        buttonY,
		RetryX:         retryX,
		RetryWidth:     retryWidth,
		RetryHeight:    retryHeight,
		RetryRadius:    retryRadius,
		XButtonSize:    xButtonSize,
		XButtonX:       xButtonX,
		XButtonCenterX: xButtonX + xButtonSize/2,
		XButtonCenterY: buttonY + retryHeight/2,
	}
}

type ColorPalette struct {
	Beige     color.NRGBA
	DarkTeal  color.NRGBA
	RedBrown  color.NRGBA
	White     color.NRGBA
	Black     color.NRGBA
	LightGreen color.NRGBA
	Cyan      color.NRGBA
}

func NewColorPalette() ColorPalette {
	return ColorPalette{
		Beige:      color.NRGBA{245, 230, 211, 255},
		DarkTeal:   color.NRGBA{61, 90, 92, 255},
		RedBrown:   color.NRGBA{200, 90, 84, 255},
		White:      color.NRGBA{255, 255, 255, 255},
		Black:      color.NRGBA{0, 0, 0, 255},
		LightGreen: color.NRGBA{237, 248, 208, 255},
		Cyan:       color.NRGBA{168, 230, 229, 255},
	}
}

type Renderer struct {
	colors ColorPalette
}

func NewRenderer() *Renderer {
	return &Renderer{
		colors: NewColorPalette(),
	}
}

func (r *Renderer) DrawBackground(screen *ebiten.Image, paddingBottom float64) {
	screen.Fill(r.colors.Black)

	var path vector.Path
	path.MoveTo(0, 0)
	path.LineTo(0, ScreenHeight-float32(paddingBottom))
	path.LineTo(ScreenWidth, ScreenHeight-float32(paddingBottom))
	path.LineTo(ScreenWidth, 0)
	path.LineTo(0, 0)

	r.fillPath(screen, path, r.colors.LightGreen)
	r.strokePath(screen, path, r.colors.Cyan, 10)
}

func (r *Renderer) DrawFruit(screen *ebiten.Image, kind assets.Kind, x, y, angle float64) {
	imgSet := assets.Get(kind)
	img := imgSet.EbitenImage
	size := img.Bounds().Size()

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(size.X)/2, -float64(size.Y)/2)
	op.GeoM.Rotate(angle)
	op.GeoM.Scale(imgSet.Scale, imgSet.Scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func (r *Renderer) DrawSpeakerButton(screen *ebiten.Image, muted bool) {
	cfg := SpeakerButtonConfig

	var icon *ebiten.Image
	if muted {
		icon = assets.GetIcon(assets.Muted)
	} else {
		icon = assets.GetIcon(assets.Speaker)
	}

	iconBounds := icon.Bounds()
	scaleX := float64(cfg.Width) / float64(iconBounds.Dx())
	scaleY := float64(cfg.Height) / float64(iconBounds.Dy())
	scale := min(scaleX, scaleY)

	scaledWidth := float64(iconBounds.Dx()) * scale
	scaledHeight := float64(iconBounds.Dy()) * scale
	offsetX := (float64(cfg.Width) - scaledWidth) / 2
	offsetY := (float64(cfg.Height) - scaledHeight) / 2

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(cfg.X)+offsetX, float64(cfg.Y)+offsetY)
	screen.DrawImage(icon, op)
}

func (r *Renderer) DrawGameOverDialog(screen *ebiten.Image, score, watermelonHits int) {
	cfg := NewDialogConfig()

	var overlayPath vector.Path
	overlayPath.MoveTo(0, 0)
	overlayPath.LineTo(ScreenWidth, 0)
	overlayPath.LineTo(ScreenWidth, ScreenHeight)
	overlayPath.LineTo(0, ScreenHeight)
	overlayPath.Close()
	r.fillPath(screen, overlayPath, color.NRGBA{255, 255, 255, 178})

	r.drawRoundedRect(screen, cfg.X, cfg.Y, cfg.Width, cfg.Height, cfg.Radius, r.colors.Beige)
	r.strokePath(screen, r.createRoundedRectPath(cfg.X, cfg.Y, cfg.Width, cfg.Height, cfg.Radius), r.colors.DarkTeal, cfg.BorderWidth)

	centerX := cfg.X + cfg.Width/2
	DrawTextCentered(screen, "GAME OVER", 42, float64(centerX), float64(cfg.Y+60), r.colors.RedBrown, true)
	DrawTextCentered(screen, "SCORE", 18, float64(centerX), float64(cfg.Y+120), r.colors.DarkTeal, true)
	DrawTextCentered(screen, fmt.Sprintf("%d", score), 60, float64(centerX), float64(cfg.Y+175), r.colors.DarkTeal, true)
	DrawTextCentered(screen, "WATERMELONS HITS", 16, float64(centerX), float64(cfg.Y+235), r.colors.DarkTeal, true)
	DrawTextCentered(screen, fmt.Sprintf("%d", watermelonHits), 60, float64(centerX), float64(cfg.Y+290), r.colors.DarkTeal, true)

	r.drawRoundedRect(screen, cfg.RetryX, cfg.ButtonY, cfg.RetryWidth, cfg.RetryHeight, cfg.RetryRadius, r.colors.RedBrown)
	DrawTextCentered(screen, "RETRY", 28, float64(cfg.RetryX+cfg.RetryWidth/2), float64(cfg.ButtonY+cfg.RetryHeight/2), r.colors.White, true)
}

func (r *Renderer) DrawTitleScreen(screen *ebiten.Image, paddingBottom float64) {
	r.DrawBackground(screen, paddingBottom)

	titleLogo := assets.GetIcon(assets.TitleLogo)
	titleLogoBounds := titleLogo.Bounds()

	const maxLogoWidth = 400.0
	logoScale := maxLogoWidth / float64(titleLogoBounds.Dx())
	if logoScale > 1.5 {
		logoScale = 1.5
	}

	scaledLogoWidth := float64(titleLogoBounds.Dx()) * logoScale
	logoX := (ScreenWidth - scaledLogoWidth) / 2
	logoY := 120.0

	logoOp := &ebiten.DrawImageOptions{}
	logoOp.Filter = ebiten.FilterLinear
	logoOp.GeoM.Scale(logoScale, logoScale)
	logoOp.GeoM.Translate(logoX, logoY)
	screen.DrawImage(titleLogo, logoOp)
}

func (r *Renderer) drawRoundedRect(screen *ebiten.Image, x, y, width, height, radius float32, clr color.NRGBA) {
	path := r.createRoundedRectPath(x, y, width, height, radius)
	r.fillPath(screen, path, clr)
}

func (r *Renderer) createRoundedRectPath(x, y, width, height, radius float32) vector.Path {
	var path vector.Path
	path.MoveTo(x+radius, y)
	path.LineTo(x+width-radius, y)
	path.ArcTo(x+width, y, x+width, y+radius, radius)
	path.LineTo(x+width, y+height-radius)
	path.ArcTo(x+width, y+height, x+width-radius, y+height, radius)
	path.LineTo(x+radius, y+height)
	path.ArcTo(x, y+height, x, y+height-radius, radius)
	path.LineTo(x, y+radius)
	path.ArcTo(x, y, x+radius, y, radius)
	path.Close()
	return path
}

func (r *Renderer) fillPath(screen *ebiten.Image, path vector.Path, clr color.NRGBA) {
	drawOp := &vector.DrawPathOptions{}
	drawOp.AntiAlias = true
	drawOp.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, drawOp)
}

func (r *Renderer) strokePath(screen *ebiten.Image, path vector.Path, clr color.NRGBA, width float32) {
	strokeOp := &vector.StrokeOptions{}
	strokeOp.Width = width
	strokeOp.LineJoin = vector.LineJoinRound

	drawOp := &vector.DrawPathOptions{}
	drawOp.AntiAlias = true
	drawOp.ColorScale.ScaleWithColor(clr)

	vector.StrokePath(screen, &path, strokeOp, drawOp)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
