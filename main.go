package main

// This is based on "jakecoffman/cp-examples/march".

import (
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
	"github.com/hajimehoshi/ebiten/v2/vector"
	assets "github.com/ponyo877/suika-shaker/assets/image"

	"github.com/jakecoffman/cp/v2"
)

var (
	bgImage       = ebiten.NewImage(100, 50)
	whiteSubImage = ebiten.NewImage(3, 3)
	score         = 0
	hiscore       = 0
	currentGame   *Game
)

const (
	screenWidth     = 480
	screenHeight    = 800
	containerHeight = 800
	paddingBottom   = 0
)

type Game struct {
	count     int
	dropCount int // Counter for auto-dropping fruits every second
	space     *cp.Space
	drawer    *ebitencp.Drawer
	next      next

	debug bool
}

type next struct {
	kind  assets.Kind
	x     float64
	y     float64
	angle float64
}

func (g *Game) Update() error {
	g.count++
	g.dropCount++

	// Auto-drop fruit every second (60 frames)
	if g.dropCount >= 60 {
		g.dropAuto()
		g.dropCount = 0
	}

	// Update gravity based on acceleration sensor
	ax, ay, _ := getAcceleration()
	// Acceleration sensor measures device acceleration, but gravity acts opposite
	// When device tilts right, sensor shows left acceleration, but gravity pulls right
	// Scale by ~50 for good game physics (typical mobile acceleration is ~9.8 m/s^2)
	gravityX := ax * 50
	gravityY := -ay * 50

	// If no acceleration data (native build or sensor not active), use default gravity
	if ax == 0 && ay == 0 {
		gravityY = 500
	}

	g.space.SetGravity(cp.Vector{X: gravityX, Y: gravityY})

	g.space.EachBody(func(body *cp.Body) {
		if body.Position().Y < screenHeight-containerHeight {
			g.space.EachShape(func(shape *cp.Shape) {
				if shape.Body().UserData != nil {
					g.space.AddPostStepCallback(removeShapeCallback, shape, nil)
				}
				hiscore = int(math.Max(float64(score), float64(hiscore)))
				score = 0
			})
			return
		}
	})
	g.next.angle += 0.01
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.drop()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.moveRight()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.moveLeft()
	}
	g.drawer.HandleMouseEvent(g.space)
	g.space.Step(1 / 60.0)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawBackground(screen)
	g.drawFruit(screen, g.next.kind, g.next.x, g.next.y-paddingBottom, g.next.angle)
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
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) moveRight() {
	if g.next.x < screenWidth-50 {
		g.next.x += 4
	}
}

func (g *Game) moveLeft() {
	if g.next.x > 40 {
		g.next.x -= 4
	}
}

func (g *Game) drop() {
	if g.count > 40 {
		k := g.next.kind
		addShapeOptions := addShapeOptions{
			kind:  g.next.kind,
			pos:   cp.Vector{X: g.next.x, Y: g.next.y},
			angle: g.next.angle,
		}
		g.space.AddPostStepCallback(addShapeCallback, k, addShapeOptions)
		g.count = 0
		g.next.kind = assets.Kind(rand.Intn(2) + int(assets.Min))
	}
}

func (g *Game) dropAuto() {
	// Generate random x position for dropping (between 50 and screenWidth-50 to avoid walls)
	randomX := float64(rand.Intn(screenWidth-100) + 50)

	k := g.next.kind
	addShapeOptions := addShapeOptions{
		kind:  g.next.kind,
		pos:   cp.Vector{X: randomX, Y: g.next.y},
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
	sop := &vector.StrokeOptions{}
	sop.Width = width
	sop.LineJoin = vector.LineJoinRound
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, sop)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(c.R) / float32(0xff)
		vs[i].ColorG = float32(c.G) / float32(0xff)
		vs[i].ColorB = float32(c.B) / float32(0xff)
		vs[i].ColorA = float32(c.A) / float32(0xff)
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.FillRule = ebiten.FillRuleFillAll
	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

func (g *Game) drawFill(screen *ebiten.Image, path vector.Path, c color.NRGBA) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(c.R) / float32(0xff)
		vs[i].ColorG = float32(c.G) / float32(0xff)
		vs[i].ColorB = float32(c.B) / float32(0xff)
		vs[i].ColorA = float32(c.A) / float32(0xff)
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.FillRule = ebiten.FillRuleFillAll
	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

func init() {
	whiteSubImage.Fill(color.White)
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

	// Set global game reference for WASM callbacks
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

	space.AddPostStepCallback(removeShapeCallback, shape, nil)
	space.AddPostStepCallback(removeShapeCallback, shape2, nil)

	score += k.Score()

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
