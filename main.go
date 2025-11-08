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
	"golang.org/x/image/font/opentype"

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
	watermelonCollisions = 0 // Track watermelon-to-watermelon collisions
	currentGame          *Game

	// Font sources for dialog
	poppinsBoldSource    *text.GoTextFaceSource
	poppinsRegularSource *text.GoTextFaceSource
)

const (
	screenWidth      = 480
	screenHeight     = 800
	containerHeight  = 800
	paddingBottom    = 0
	maxSpawnFailures = 3  // Maximum consecutive spawn failures before game over
	spawnCheckRadius = 40 // Collision check radius for spawn position (Grape radius)
)

type Game struct {
	count          int
	dropCount      int // Counter for auto-dropping fruits every second
	spawnFailCount int // Counter for consecutive spawn failures
	space          *cp.Space
	drawer         *ebitencp.Drawer
	next           next

	debug               bool
	gameOver            bool
	gameOverSE          bool          // Flag to ensure game over sound plays only once
	muted               bool          // Mute state for audio
	showGameOverDialog  bool          // Flag to show game over dialog
	showTitleScreen     bool          // Flag to show title screen
	gameOverScreenshot  *ebiten.Image // Screenshot captured at game over
	finalScore          int           // Score at game over
	finalWatermelonHits int           // Watermelon collisions at game over
}

type next struct {
	kind  assets.Kind
	x     float64
	y     float64
	angle float64
}

func (g *Game) Update() error {
	g.count++

	// Handle title screen
	if g.showTitleScreen {
		// Title screen input is handled by JavaScript (setupTapEvent in index.html)
		// to maintain user gesture context for iOS motion sensor permissions
		// and browser autoplay policy for audio.
		// JavaScript will call startGameFromJS() to start the game.
		return nil // Skip game logic when showing title screen
	}

	g.dropCount++

	// Auto-drop fruit every second (45 frames) - only if not showing game over dialog
	if !g.showGameOverDialog && g.dropCount >= 45 {
		g.dropAuto()
		g.dropCount = 0
	}

	// Update gravity based on acceleration sensor
	ax, ay, _ := getAcceleration()
	// Note: JavaScript side (index.html) already inverts values for iOS/Android
	// to normalize platform differences per W3C DeviceMotionEvent spec
	// Acceleration sensor measures device acceleration, but gravity acts opposite
	// When device tilts right, sensor shows left acceleration, but gravity pulls right
	// Scale by 100 for good game physics (typical mobile acceleration is ~9.8 m/s^2)
	gravityX := ax * 100
	gravityY := -ay * 100 // Y-axis negation converts sensor acceleration to gravity direction

	// If no acceleration data (native build or sensor not active), use default gravity
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

	// Handle game over state
	if g.gameOver && !g.showGameOverDialog {
		// Show dialog instead of immediate reset
		g.showGameOverDialog = true
		g.finalScore = score
		g.finalWatermelonHits = watermelonCollisions
		hiscore = int(math.Max(float64(score), float64(hiscore)))

		// Stop all fruits from moving
		g.space.EachBody(func(body *cp.Body) {
			if body.UserData != nil {
				body.SetVelocity(0, 0)
				body.SetAngularVelocity(0)
			}
		})
	}

	g.next.angle += 0.01

	// Handle speaker button click/touch
	const (
		buttonX    = screenWidth - 60
		buttonY    = 10
		buttonSize = 50
	)

	// Mouse click detection
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= buttonX && x <= buttonX+buttonSize && y >= buttonY && y <= buttonY+buttonSize {
			g.muted = !g.muted
			sound.SetMuted(g.muted)
		}
	}

	// Touch detection (for mobile/WASM)
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	for _, id := range touchIDs {
		x, y := ebiten.TouchPosition(id)
		if x >= buttonX && x <= buttonX+buttonSize && y >= buttonY && y <= buttonY+buttonSize {
			g.muted = !g.muted
			sound.SetMuted(g.muted)
		}
	}

	// Handle game over dialog button clicks
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

		// Mouse click detection
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			// Check Retry button
			if x >= int(retryX) && x <= int(retryX+retryWidth) &&
				y >= int(buttonY) && y <= int(buttonY+retryHeight) {
				g.resetGame()
			}
			// Share button is now handled by HTML overlay (no Go-side handling needed)
		}

		// Touch detection (for mobile/WASM)
		touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
		for _, id := range touchIDs {
			x, y := ebiten.TouchPosition(id)
			// Check Retry button
			if x >= int(retryX) && x <= int(retryX+retryWidth) &&
				y >= int(buttonY) && y <= int(buttonY+retryHeight) {
				g.resetGame()
			}
			// Share button is now handled by HTML overlay (no Go-side handling needed)
		}
	}

	g.drawer.HandleMouseEvent(g.space)
	g.space.Step(1 / 60.0)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Show title screen instead of game if showTitleScreen is true
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
		"FPS: %0.2f\nScore: %d\nHiScore: %d",
		ebiten.ActualFPS(),
		score,
		hiscore,
	))

	// Draw speaker/mute button in top-right corner
	g.drawSpeakerButton(screen)

	// Draw game over dialog if active
	if g.showGameOverDialog {
		g.drawGameOverDialog(screen)

		// Capture screenshot after dialog is drawn (on first frame)
		if g.gameOverScreenshot == nil {
			// Create a copy of the current screen with dialog for screenshot
			g.gameOverScreenshot = ebiten.NewImage(screenWidth, screenHeight)
			g.gameOverScreenshot.DrawImage(screen, nil)

			// Show share button with screenshot data (WASM only)
			shareGameResultToX(g.gameOverScreenshot, g.finalScore, g.finalWatermelonHits)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) dropAuto() {
	// Generate random x position for dropping (between 50 and screenWidth-50 to avoid walls)
	// randomX := float64(rand.Intn(screenWidth-100) + 50)
	// randomY := g.next.y

	// Check if we can spawn at this position
	if !g.canSpawnAt(g.next.x, g.next.y, spawnCheckRadius) {
		// Cannot spawn - increment fail counter
		g.spawnFailCount++
		if g.spawnFailCount >= maxSpawnFailures {
			// Trigger game over
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

	// Reset fail counter on successful spawn
	g.spawnFailCount = 0

	k := g.next.kind
	addShapeOptions := addShapeOptions{
		kind:  g.next.kind,
		pos:   cp.Vector{X: g.next.x, Y: g.next.y},
		angle: g.next.angle,
	}
	g.space.AddPostStepCallback(addShapeCallback, k, addShapeOptions)

	// Set next fruit with random position
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
	boldFont, err := opentype.Parse(poppinsBoldTTF)
	if err != nil {
		log.Fatal(err)
	}
	poppinsBoldSource, err = text.NewGoTextFaceSource(bytes.NewReader(poppinsBoldTTF))
	if err != nil {
		log.Fatal(err)
	}

	regularFont, err := opentype.Parse(poppinsRegularTTF)
	if err != nil {
		log.Fatal(err)
	}
	poppinsRegularSource, err = text.NewGoTextFaceSource(bytes.NewReader(poppinsRegularTTF))
	if err != nil {
		log.Fatal(err)
	}

	_, _ = boldFont, regularFont // Suppress unused variable warnings
}

func main() {
	// Setup WASM callbacks before starting the game
	setupWASMCallbacks()

	// chipmunk init
	space := cp.NewSpace()
	space.Iterations = 30
	// Initial gravity will be set in Update() based on sensor data
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

	// ebitengine init
	bgImage.Fill(color.Black)

	game := &Game{}
	game.space = space
	game.drawer = ebitencp.NewDrawer(screenWidth, screenHeight)
	game.drawer.FlipYAxis = true
	game.next = next{kind: assets.Grape, x: screenWidth / 2, y: screenHeight - containerHeight + 10, angle: 0}
	game.showTitleScreen = true // Show title screen at startup

	// Set global game reference for WASM callbacks
	currentGame = game

	// Don't start background music here - it will be started when user presses START button

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

	// Ensure collision detection is active immediately by reindexing the body's shapes
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

	// Check if both shapes are watermelons (track watermelon collisions)
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

	// Play sound effect based on fruit type
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

// canSpawnAt checks if a fruit can be spawned at the given position without colliding
func (g *Game) canSpawnAt(x, y, radius float64) bool {
	// Use PointQueryNearest to find the nearest shape within radius
	info := g.space.PointQueryNearest(cp.Vector{X: x, Y: y}, radius, cp.ShapeFilter{})

	// If no shape found within radius, position is clear
	if info.Shape == nil {
		return true
	}

	// Check if the nearest shape is a fruit (not a wall)
	// Walls have nil UserData, fruits have Kind as UserData
	if info.Shape.Body().UserData != nil {
		return false // Collision with fruit detected
	}

	return true // Only walls nearby, position is OK for spawning
}

// countFruits counts the total number of fruits currently on screen
func (g *Game) countFruits() int {
	count := 0
	g.space.EachBody(func(body *cp.Body) {
		if body.UserData != nil {
			count++
		}
	})
	return count
}

// resetGame resets the game state and starts a new game
func (g *Game) resetGame() {
	// Remove all fruits from the game
	g.space.EachShape(func(shape *cp.Shape) {
		if shape.Body().UserData != nil {
			g.space.AddPostStepCallback(removeShapeCallback, shape, nil)
		}
	})

	// Reset game state
	score = 0
	watermelonCollisions = 0
	g.gameOver = false
	g.gameOverSE = false
	g.showGameOverDialog = false
	g.gameOverScreenshot = nil
	g.finalScore = 0
	g.finalWatermelonHits = 0
	g.spawnFailCount = 0

	// Hide share button (WASM only)
	hideShareButton()

	// Restart background music only if not muted
	if !g.muted {
		sound.StartBackgroundMusic()
	}
}

// shareToX shares the game result to X (Twitter) with screenshot
func (g *Game) shareToX() {
	// This will be implemented with WASM-specific code
	shareGameResultToX(g.gameOverScreenshot, g.finalScore, g.finalWatermelonHits)
}

// drawTextCentered draws text centered horizontally at the given position
func drawTextCentered(screen *ebiten.Image, str string, source *text.GoTextFaceSource, size float64, x, y float64, clr color.Color) {
	face := &text.GoTextFace{
		Source: source,
		Size:   size,
	}

	// Calculate text width to center it
	textWidth, textHeight := text.Measure(str, face, 0)

	op := &text.DrawOptions{}
	op.GeoM.Translate(x-textWidth/2, y-textHeight/2)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, face, op)
}

// drawRoundedRect draws a rounded rectangle with the given parameters
func (g *Game) drawRoundedRect(screen *ebiten.Image, x, y, width, height, radius float32, clr color.NRGBA) {
	var path vector.Path

	// Start from top-left corner (after radius)
	path.MoveTo(x+radius, y)
	// Top edge
	path.LineTo(x+width-radius, y)
	// Top-right arc
	path.ArcTo(x+width, y, x+width, y+radius, radius)
	// Right edge
	path.LineTo(x+width, y+height-radius)
	// Bottom-right arc
	path.ArcTo(x+width, y+height, x+width-radius, y+height, radius)
	// Bottom edge
	path.LineTo(x+radius, y+height)
	// Bottom-left arc
	path.ArcTo(x, y+height, x, y+height-radius, radius)
	// Left edge
	path.LineTo(x, y+radius)
	// Top-left arc
	path.ArcTo(x, y, x+radius, y, radius)
	path.Close()

	g.drawFill(screen, path, clr)
}

// drawGameOverDialog draws the game over dialog matching the design image
func (g *Game) drawGameOverDialog(screen *ebiten.Image) {
	const (
		dialogWidth  = 340
		dialogHeight = 440
		dialogX      = (screenWidth - dialogWidth) / 2
		dialogY      = (screenHeight - dialogHeight) / 2
		borderWidth  = 10
		radius       = 25
	)

	// Colors from the design
	beigeColor := color.NRGBA{245, 230, 211, 255}  // #F5E6D3
	darkTealColor := color.NRGBA{61, 90, 92, 255}  // #3D5A5C
	redBrownColor := color.NRGBA{200, 90, 84, 255} // #C85A54
	whiteColor := color.NRGBA{255, 255, 255, 255}

	// Draw semi-transparent white overlay (full screen)
	var overlayPath vector.Path
	overlayPath.MoveTo(0, 0)
	overlayPath.LineTo(screenWidth, 0)
	overlayPath.LineTo(screenWidth, screenHeight)
	overlayPath.LineTo(0, screenHeight)
	overlayPath.Close()

	g.drawFill(screen, overlayPath, color.NRGBA{255, 255, 255, 178}) // 0.7 alpha = 178

	// Draw beige background with dark teal border
	g.drawRoundedRect(screen, dialogX, dialogY, dialogWidth, dialogHeight, radius, beigeColor)

	// Draw border (stroke)
	g.drawLine(screen, g.createRoundedRectPath(dialogX, dialogY, dialogWidth, dialogHeight, radius), darkTealColor, borderWidth)

	// Draw text content
	centerX := dialogX + dialogWidth/2

	// "GAME OVER" - red/brown, large
	drawTextCentered(screen, "GAME OVER", poppinsBoldSource, 42, float64(centerX), float64(dialogY+60), redBrownColor)

	// "SCORE" label - dark teal, small
	drawTextCentered(screen, "SCORE", poppinsBoldSource, 18, float64(centerX), float64(dialogY+120), darkTealColor)

	// Score value - dark teal, large
	scoreStr := fmt.Sprintf("%d", g.finalScore)
	drawTextCentered(screen, scoreStr, poppinsBoldSource, 60, float64(centerX), float64(dialogY+175), darkTealColor)

	// "WATERMELONS DESTROYED" label - dark teal, small
	drawTextCentered(screen, "WATERMELONS HITS", poppinsBoldSource, 16, float64(centerX), float64(dialogY+235), darkTealColor)

	// Watermelon count - dark teal, large
	watermelonStr := fmt.Sprintf("%d", g.finalWatermelonHits)
	drawTextCentered(screen, watermelonStr, poppinsBoldSource, 60, float64(centerX), float64(dialogY+290), darkTealColor)

	// Button layout (RETRY button only, share button is HTML overlay)
	const (
		buttonY     = dialogY + dialogHeight - 85
		retryWidth  = 230
		retryHeight = 50
		retryRadius = 15
		retryX      = dialogX + 25
	)

	// Draw RETRY button (red/brown rounded rectangle)
	g.drawRoundedRect(screen, retryX, buttonY, retryWidth, retryHeight, retryRadius, redBrownColor)

	// RETRY text (white, centered)
	drawTextCentered(screen, "RETRY", poppinsBoldSource, 28, float64(retryX+retryWidth/2), float64(buttonY+retryHeight/2), whiteColor)
}

// createRoundedRectPath creates a path for a rounded rectangle
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

// drawTitleScreen draws the title screen with logo and start button
func (g *Game) drawTitleScreen(screen *ebiten.Image) {
	// Colors matching the game design
	// redBrownColor := color.NRGBA{200, 90, 84, 255} // #C85A54
	// whiteColor := color.NRGBA{255, 255, 255, 255}

	// Draw same background as game screen
	g.drawBackground(screen)

	// Get title logo
	titleLogo := assets.GetIcon(assets.TitleLogo)
	titleLogoBounds := titleLogo.Bounds()

	// Calculate logo scale and position (centered, upper part of screen)
	const maxLogoWidth = 400.0
	logoScale := maxLogoWidth / float64(titleLogoBounds.Dx())
	if logoScale > 1.5 {
		logoScale = 1.5 // Cap the scale to avoid too large logo
	}

	scaledLogoWidth := float64(titleLogoBounds.Dx()) * logoScale
	logoX := (screenWidth - scaledLogoWidth) / 2
	logoY := 120.0

	// Draw title logo
	logoOp := &ebiten.DrawImageOptions{}
	logoOp.Filter = ebiten.FilterLinear
	logoOp.GeoM.Scale(logoScale, logoScale)
	logoOp.GeoM.Translate(logoX, logoY)
	screen.DrawImage(titleLogo, logoOp)

	// // START button dimensions and position (matching RETRY button style)
	// const (
	// 	buttonWidth  = 230
	// 	buttonHeight = 50
	// 	buttonRadius = 15
	// )
	// buttonX := float32((screenWidth - buttonWidth) / 2)
	// buttonY := float32(600)

	// // Draw START button (red/brown rounded rectangle, matching RETRY button)
	// g.drawRoundedRect(screen, buttonX, buttonY, buttonWidth, buttonHeight, buttonRadius, redBrownColor)

	// // START text (white, centered, matching RETRY button text)
	// drawTextCentered(screen, "START", poppinsBoldSource, 28, float64(buttonX+buttonWidth/2), float64(buttonY+buttonHeight/2), whiteColor)
}

// drawSpeakerButton draws the mute/unmute button in the top-right corner using images
func (g *Game) drawSpeakerButton(screen *ebiten.Image) {
	const (
		buttonX    = screenWidth - 60
		buttonY    = 10
		buttonSize = 50
	)

	// Choose the appropriate icon
	var icon *ebiten.Image
	if g.muted {
		icon = assets.GetIcon(assets.Muted)
	} else {
		icon = assets.GetIcon(assets.Speaker)
	}

	// Calculate scale to fit the icon in the button size
	iconBounds := icon.Bounds()
	scaleX := float64(buttonSize) / float64(iconBounds.Dx())
	scaleY := float64(buttonSize) / float64(iconBounds.Dy())
	scale := math.Min(scaleX, scaleY)

	// Center the icon
	scaledWidth := float64(iconBounds.Dx()) * scale
	scaledHeight := float64(iconBounds.Dy()) * scale
	offsetX := (float64(buttonSize) - scaledWidth) / 2
	offsetY := (float64(buttonSize) - scaledHeight) / 2

	// Draw the icon
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(buttonX)+offsetX, float64(buttonY)+offsetY)
	screen.DrawImage(icon, op)
}
