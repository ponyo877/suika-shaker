package main

// This is based on "jakecoffman/cp-examples/march".

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"

	"github.com/demouth/ebitencp"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	assets "github.com/ponyo877/suika-shaker/assets/image"
	"github.com/ponyo877/suika-shaker/assets/sound"

	"github.com/jakecoffman/cp/v2"
)

//go:embed assets/fonts/Poppins-Bold.ttf
var poppinsBoldTTF []byte

//go:embed assets/fonts/Poppins-Regular.ttf
var poppinsRegularTTF []byte

var (
	bgImage              = ebiten.NewImage(100, 50)
	score                = 0
	hiscore              = 0
	watermelonCollisions = 0
	currentGame          *Game

	poppinsBoldSource    *text.GoTextFaceSource
	poppinsRegularSource *text.GoTextFaceSource
)

const (
	screenWidth      = 480
	screenHeight     = 800
	containerHeight  = 800
	paddingBottom    = 0
	maxSpawnFailures = 3
	spawnCheckRadius = 40
)

type Game struct {
	count          int
	dropCount      int
	spawnFailCount int
	space          *cp.Space
	drawer         *ebitencp.Drawer
	next           next

	debug               bool
	gameOver            bool
	gameOverSE          bool
	muted               bool
	showGameOverDialog  bool
	showTitleScreen     bool
	gameOverScreenshot  *ebiten.Image
	finalScore          int
	finalWatermelonHits int
}

type next struct {
	kind  assets.Kind
	x     float64
	y     float64
	angle float64
}

func (g *Game) Update() error {
	g.count++

	if g.showTitleScreen {
		return nil
	}

	g.dropCount++

	if !g.showGameOverDialog && g.dropCount >= 45 {
		g.dropAuto()
		g.dropCount = 0
	}

	ax, ay, _ := getAcceleration()
	gravityX := ax * 100
	gravityY := -ay * 100

	if ax == 0 && ay == 0 {
		gravityY = 500
	}

	g.space.SetGravity(cp.Vector{X: gravityX, Y: gravityY})

	g.space.EachBody(func(body *cp.Body) {
		x, y := body.Position().X, body.Position().Y
		if 0 > x || x > float64(screenWidth) || 0 > y || y > float64(screenHeight) {
			if !g.gameOver {
				g.gameOver = true
			}
			if !g.gameOverSE {
				g.gameOverSE = true
				sound.PlayGameOver()
			}
		}
	})

	if g.gameOver && !g.showGameOverDialog {
		g.showGameOverDialog = true
		g.finalScore = score
		g.finalWatermelonHits = watermelonCollisions
		hiscore = int(math.Max(float64(score), float64(hiscore)))

		g.space.EachBody(func(body *cp.Body) {
			if body.UserData != nil {
				body.SetVelocity(0, 0)
				body.SetAngularVelocity(0)
			}
		})
	}

	g.next.angle += 0.01

	const (
		buttonX    = screenWidth - 60
		buttonY    = 10
		buttonSize = 50
	)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= buttonX && x <= buttonX+buttonSize && y >= buttonY && y <= buttonY+buttonSize {
			g.muted = !g.muted
			sound.SetMuted(g.muted)
		}
	}

	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	for _, id := range touchIDs {
		x, y := ebiten.TouchPosition(id)
		if x >= buttonX && x <= buttonX+buttonSize && y >= buttonY && y <= buttonY+buttonSize {
			g.muted = !g.muted
			sound.SetMuted(g.muted)
		}
	}

	if g.showGameOverDialog {
		const (
			dialogWidth  = 340
			dialogHeight = 440
			dialogX      = (screenWidth - dialogWidth) / 2
			dialogY      = (screenHeight - dialogHeight) / 2

			buttonY        = dialogY + dialogHeight - 85
			retryWidth     = 230
			retryHeight    = 50
			retryX         = dialogX + 25
			xButtonSize    = 60
			xButtonX       = retryX + retryWidth + 15
			xButtonCenterX = xButtonX + xButtonSize/2
			xButtonCenterY = buttonY + retryHeight/2
		)

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			if x >= int(retryX) && x <= int(retryX+retryWidth) &&
				y >= int(buttonY) && y <= int(buttonY+retryHeight) {
				g.resetGame()
			}
		}

		touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
		for _, id := range touchIDs {
			x, y := ebiten.TouchPosition(id)
			if x >= int(retryX) && x <= int(retryX+retryWidth) &&
				y >= int(buttonY) && y <= int(buttonY+retryHeight) {
				g.resetGame()
			}
		}
	}

	g.drawer.HandleMouseEvent(g.space)
	g.space.Step(1 / 60.0)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.showTitleScreen {
		g.drawTitleScreen(screen)
		return
	}

	g.drawBackground(screen)
	g.space.EachShape(func(shape *cp.Shape) {
		switch shape.Class.(type) {
		case *cp.PolyShape:
			circle := shape.Class.(*cp.PolyShape)
			vec := circle.Body().Position()
			g.drawFruit(screen, circle.Body().UserData.(assets.Kind), vec.X, vec.Y-paddingBottom, circle.Body().Angle())
		}
	})
	if g.debug {
		cp.DrawSpace(g.space, g.drawer.WithScreen(screen))
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf(
		"FPS: %0.2f  The Go gopher was designed by Renee French.\nScore: %d\nHiScore: %d",
		ebiten.ActualFPS(),
		score,
		hiscore,
	))

	// Draw speaker/mute button in top-right corner
	g.drawSpeakerButton(screen)

	if g.showGameOverDialog {
		g.drawGameOverDialog(screen)

		if g.gameOverScreenshot == nil {
			g.gameOverScreenshot = ebiten.NewImage(screenWidth, screenHeight)
			g.gameOverScreenshot.DrawImage(screen, nil)

			shareGameResultToX(g.gameOverScreenshot, g.finalScore, g.finalWatermelonHits)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) dropAuto() {
	if !g.canSpawnAt(g.next.x, g.next.y, spawnCheckRadius) {
		g.spawnFailCount++
		if g.spawnFailCount >= maxSpawnFailures {
			if !g.gameOver {
				g.gameOver = true
			}
			if !g.gameOverSE {
				g.gameOverSE = true
				sound.PlayGameOver()
				sound.StopBackgroundMusic()
			}
		}
		return
	}

	g.spawnFailCount = 0

	k := g.next.kind
	addShapeOptions := addShapeOptions{
		kind:  g.next.kind,
		pos:   cp.Vector{X: g.next.x, Y: g.next.y},
		angle: g.next.angle,
	}
	g.space.AddPostStepCallback(addShapeCallback, k, addShapeOptions)

	g.next.kind = assets.Kind(rand.Intn(2) + int(assets.Min))
	g.next.x = float64(rand.Intn(screenWidth-100) + 50)
	g.next.y = float64(rand.Intn(screenHeight-100) + 50)
	g.next.angle = rand.Float64() * 2 * math.Pi
}

func (g *Game) drawBackground(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})
	screen.DrawImage(bgImage, nil)

	var path vector.Path

	path = vector.Path{}
	path.MoveTo(0, 0)
	path.LineTo(0, screenHeight-paddingBottom)
	path.LineTo(screenWidth, screenHeight-paddingBottom)
	path.LineTo(screenWidth, 0)
	path.LineTo(0, 0)
	g.drawFill(screen, path, color.NRGBA{0xed, 0xf8, 0xd0, 0xff})
	g.drawLine(screen, path, color.NRGBA{0xa8, 0xe6, 0xe5, 0xff}, 10)
}

func (g *Game) drawLine(screen *ebiten.Image, path vector.Path, c color.NRGBA, width float32) {
	strokeOp := &vector.StrokeOptions{}
	strokeOp.Width = width
	strokeOp.LineJoin = vector.LineJoinRound

	drawOp := &vector.DrawPathOptions{}
	drawOp.AntiAlias = true
	drawOp.ColorScale.ScaleWithColor(c)

	vector.StrokePath(screen, &path, strokeOp, drawOp)
}

func (g *Game) drawFill(screen *ebiten.Image, path vector.Path, c color.NRGBA) {
	drawOp := &vector.DrawPathOptions{}
	drawOp.AntiAlias = true
	drawOp.ColorScale.ScaleWithColor(c)

	vector.FillPath(screen, &path, nil, drawOp)
}

func init() {
	// Load fonts
	var err error
	poppinsBoldSource, err = text.NewGoTextFaceSource(bytes.NewReader(poppinsBoldTTF))
	if err != nil {
		log.Fatal(err)
	}

	poppinsRegularSource, err = text.NewGoTextFaceSource(bytes.NewReader(poppinsRegularTTF))
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	setupWASMCallbacks()

	space := cp.NewSpace()
	space.Iterations = 30
	space.SetGravity(cp.Vector{X: 0, Y: 500})
	space.SleepTimeThreshold = 0.5
	space.SetDamping(1)

	walls := []cp.Vector{
		{X: 0, Y: 0}, {X: 0, Y: screenHeight},
		{X: screenWidth, Y: 0}, {X: screenWidth, Y: screenHeight},
		{X: 0, Y: screenHeight}, {X: screenWidth, Y: screenHeight},
		{X: 0, Y: 0}, {X: screenWidth, Y: 0},
	}
	for i := 0; i < len(walls)-1; i += 2 {
		shape := space.AddShape(cp.NewSegment(space.StaticBody, walls[i], walls[i+1], 1))
		shape.SetElasticity(0.6)
		shape.SetFriction(0.4)
	}

	assets.ForEach(func(i assets.Kind, is assets.ImageSet) {
		ct := cp.CollisionType(i)
		space.NewCollisionHandler(ct, ct).BeginFunc = BeginFunc
	})

	bgImage.Fill(color.Black)

	game := &Game{}
	game.space = space
	game.drawer = ebitencp.NewDrawer(screenWidth, screenHeight)
	game.drawer.FlipYAxis = true
	game.next = next{kind: assets.Grape, x: screenWidth / 2, y: screenHeight - containerHeight + 10, angle: 0}
	game.showTitleScreen = true

	currentGame = game

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Suika Shaker")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func addFruit(space *cp.Space, k assets.Kind, position cp.Vector, angle float64) {
	if !assets.Exists(k) {
		return
	}
	imgSet := assets.Get(k)

	body := space.AddBody(cp.NewBody(0, cp.MomentForPoly(10, len(imgSet.Vectors), imgSet.Vectors, cp.Vector{}, 1)))
	body.SetPosition(position)
	body.SetAngle(angle)
	body.UserData = k
	fruit := space.AddShape(cp.NewPolyShape(body, len(imgSet.Vectors), imgSet.Vectors, cp.NewTransformIdentity(), 0))
	body.SetMass(fruit.Area() * 0.001)
	fruit.SetElasticity(0.2)
	fruit.SetFriction(0.9)
	fruit.SetCollisionType(cp.CollisionType(k))

	body.Activate()
	space.ReindexShape(fruit)
}

func (g *Game) drawFruit(screen *ebiten.Image, kind assets.Kind, x, y, angle float64) {
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

type addShapeOptions struct {
	kind  assets.Kind
	pos   cp.Vector
	angle float64
}

func addShapeCallback(space *cp.Space, key interface{}, data interface{}) {
	var opt addShapeOptions
	if i, ok := data.(addShapeOptions); ok {
		opt = i
	} else {
		return
	}
	addFruit(space, opt.kind, opt.pos, opt.angle)
}

func removeShapeCallback(space *cp.Space, key interface{}, data interface{}) {
	var s *cp.Shape
	var ok bool
	if s, ok = key.(*cp.Shape); !ok {
		return
	}
	space.RemoveBody(s.Body())
	space.RemoveShape(s)
}

func BeginFunc(arb *cp.Arbiter, space *cp.Space, data interface{}) bool {
	shape, shape2 := arb.Shapes()

	var k assets.Kind
	if ud, ok := shape.Body().UserData.(assets.Kind); ok {
		k = ud
	} else {
		return false
	}

	var k2 assets.Kind
	if ud2, ok := shape2.Body().UserData.(assets.Kind); ok {
		k2 = ud2
	}
	if k == assets.Watermelon && k2 == assets.Watermelon {
		watermelonCollisions++
	}

	space.AddPostStepCallback(removeShapeCallback, shape, nil)
	space.AddPostStepCallback(removeShapeCallback, shape2, nil)

	score += k.Score()

	if k == assets.Melon || k == assets.Watermelon {
		sound.PlaySuikaJoin()
	} else {
		sound.PlayJoin()
	}

	if hasNext, kk := k.Next(); hasNext {
		k = kk
	} else {
		return false
	}
	sp := shape.Body().Position().Clone()
	sp.Sub(shape2.Body().Position()).Mult(0.5).Add(shape2.Body().Position())
	a := (shape.Body().Angle() + shape2.Body().Angle()) / 2
	addShapeOptions := addShapeOptions{
		kind:  k,
		pos:   sp,
		angle: a,
	}
	space.AddPostStepCallback(addShapeCallback, k, addShapeOptions)
	return false
}

func (g *Game) canSpawnAt(x, y, radius float64) bool {
	info := g.space.PointQueryNearest(cp.Vector{X: x, Y: y}, radius, cp.ShapeFilter{})

	if info.Shape == nil {
		return true
	}

	if info.Shape.Body().UserData != nil {
		return false
	}

	return true
}

func (g *Game) resetGame() {
	g.space.EachShape(func(shape *cp.Shape) {
		if shape.Body().UserData != nil {
			g.space.AddPostStepCallback(removeShapeCallback, shape, nil)
		}
	})

	score = 0
	watermelonCollisions = 0
	g.gameOver = false
	g.gameOverSE = false
	g.showGameOverDialog = false
	g.gameOverScreenshot = nil
	g.finalScore = 0
	g.finalWatermelonHits = 0
	g.spawnFailCount = 0

	hideShareButton()

	if !g.muted {
		sound.StartBackgroundMusic()
	}
}

func drawTextCentered(screen *ebiten.Image, str string, source *text.GoTextFaceSource, size float64, x, y float64, clr color.Color) {
	face := &text.GoTextFace{
		Source: source,
		Size:   size,
	}

	textWidth, textHeight := text.Measure(str, face, 0)

	op := &text.DrawOptions{}
	op.GeoM.Translate(x-textWidth/2, y-textHeight/2)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, face, op)
}

func (g *Game) drawRoundedRect(screen *ebiten.Image, x, y, width, height, radius float32, clr color.NRGBA) {
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

	g.drawFill(screen, path, clr)
}

func (g *Game) drawGameOverDialog(screen *ebiten.Image) {
	const (
		dialogWidth  = 340
		dialogHeight = 440
		dialogX      = (screenWidth - dialogWidth) / 2
		dialogY      = (screenHeight - dialogHeight) / 2
		borderWidth  = 10
		radius       = 25
	)

	beigeColor := color.NRGBA{245, 230, 211, 255}  // #F5E6D3
	darkTealColor := color.NRGBA{61, 90, 92, 255}  // #3D5A5C
	redBrownColor := color.NRGBA{200, 90, 84, 255} // #C85A54
	whiteColor := color.NRGBA{255, 255, 255, 255}

	var overlayPath vector.Path
	overlayPath.MoveTo(0, 0)
	overlayPath.LineTo(screenWidth, 0)
	overlayPath.LineTo(screenWidth, screenHeight)
	overlayPath.LineTo(0, screenHeight)
	overlayPath.Close()

	g.drawFill(screen, overlayPath, color.NRGBA{255, 255, 255, 178})

	g.drawRoundedRect(screen, dialogX, dialogY, dialogWidth, dialogHeight, radius, beigeColor)

	g.drawLine(screen, g.createRoundedRectPath(dialogX, dialogY, dialogWidth, dialogHeight, radius), darkTealColor, borderWidth)

	centerX := dialogX + dialogWidth/2

	drawTextCentered(screen, "GAME OVER", poppinsBoldSource, 42, float64(centerX), float64(dialogY+60), redBrownColor)

	drawTextCentered(screen, "SCORE", poppinsBoldSource, 18, float64(centerX), float64(dialogY+120), darkTealColor)

	scoreStr := fmt.Sprintf("%d", g.finalScore)
	drawTextCentered(screen, scoreStr, poppinsBoldSource, 60, float64(centerX), float64(dialogY+175), darkTealColor)

	drawTextCentered(screen, "WATERMELONS HITS", poppinsBoldSource, 16, float64(centerX), float64(dialogY+235), darkTealColor)

	watermelonStr := fmt.Sprintf("%d", g.finalWatermelonHits)
	drawTextCentered(screen, watermelonStr, poppinsBoldSource, 60, float64(centerX), float64(dialogY+290), darkTealColor)

	const (
		buttonY     = dialogY + dialogHeight - 85
		retryWidth  = 230
		retryHeight = 50
		retryRadius = 15
		retryX      = dialogX + 25
	)

	g.drawRoundedRect(screen, retryX, buttonY, retryWidth, retryHeight, retryRadius, redBrownColor)

	drawTextCentered(screen, "RETRY", poppinsBoldSource, 28, float64(retryX+retryWidth/2), float64(buttonY+retryHeight/2), whiteColor)
}

func (g *Game) createRoundedRectPath(x, y, width, height, radius float32) vector.Path {
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

func (g *Game) drawTitleScreen(screen *ebiten.Image) {
	g.drawBackground(screen)

	titleLogo := assets.GetIcon(assets.TitleLogo)
	titleLogoBounds := titleLogo.Bounds()

	const maxLogoWidth = 400.0
	logoScale := maxLogoWidth / float64(titleLogoBounds.Dx())
	if logoScale > 1.5 {
		logoScale = 1.5
	}

	scaledLogoWidth := float64(titleLogoBounds.Dx()) * logoScale
	logoX := (screenWidth - scaledLogoWidth) / 2
	logoY := 120.0

	logoOp := &ebiten.DrawImageOptions{}
	logoOp.Filter = ebiten.FilterLinear
	logoOp.GeoM.Scale(logoScale, logoScale)
	logoOp.GeoM.Translate(logoX, logoY)
	screen.DrawImage(titleLogo, logoOp)
}

func (g *Game) drawSpeakerButton(screen *ebiten.Image) {
	const (
		buttonX    = screenWidth - 60
		buttonY    = 10
		buttonSize = 50
	)

	var icon *ebiten.Image
	if g.muted {
		icon = assets.GetIcon(assets.Muted)
	} else {
		icon = assets.GetIcon(assets.Speaker)
	}

	iconBounds := icon.Bounds()
	scaleX := float64(buttonSize) / float64(iconBounds.Dx())
	scaleY := float64(buttonSize) / float64(iconBounds.Dy())
	scale := math.Min(scaleX, scaleY)

	scaledWidth := float64(iconBounds.Dx()) * scale
	scaledHeight := float64(iconBounds.Dy()) * scale
	offsetX := (float64(buttonSize) - scaledWidth) / 2
	offsetY := (float64(buttonSize) - scaledHeight) / 2

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(buttonX)+offsetX, float64(buttonY)+offsetY)
	screen.DrawImage(icon, op)
}
