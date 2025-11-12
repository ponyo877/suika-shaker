package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/demouth/ebitencp"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp/v2"
	assets "github.com/ponyo877/suika-shaker/assets/image"
	"github.com/ponyo877/suika-shaker/assets/sound"
	"github.com/ponyo877/suika-shaker/internal/gamestate"
	"github.com/ponyo877/suika-shaker/internal/input"
	"github.com/ponyo877/suika-shaker/internal/physics"
	"github.com/ponyo877/suika-shaker/internal/ui"
)

var currentGame *Game

type Game struct {
	state          *gamestate.State
	physicsManager *physics.Manager
	renderer       *ui.Renderer
	inputHandler   *input.Handler
	drawer         *ebitencp.Drawer
	debug          bool
}

func NewGame() *Game {
	physManager := physics.NewManager()

	assets.ForEach(func(kind assets.Kind, _ assets.ImageSet) {
		ct := cp.CollisionType(kind)
		physManager.GetSpace().NewCollisionHandler(ct, ct).BeginFunc = createCollisionHandler(physManager)
	})

	state := gamestate.NewState()
	state.NextFruit = gamestate.NextFruit{
		Kind:  assets.Grape,
		X:     ui.ScreenWidth / 2,
		Y:     ui.ScreenHeight - physics.ContainerHeight + 10,
		Angle: 0,
	}

	drawer := ebitencp.NewDrawer(ui.ScreenWidth, ui.ScreenHeight)
	drawer.FlipYAxis = true

	return &Game{
		state:          state,
		physicsManager: physManager,
		renderer:       ui.NewRenderer(),
		inputHandler:   input.NewHandler(),
		drawer:         drawer,
		debug:          false,
	}
}

func (g *Game) Update() error {
	g.state.IncrementCount()

	if g.state.ShowTitleScreen {
		return nil
	}

	g.updatePhysics()
	g.handleInput()
	g.updateDropLogic()
	g.updateAnimation()
	g.checkGameOver()

	return nil
}

func (g *Game) updatePhysics() {
	ax, ay, _ := getAcceleration()
	gravityX := ax * 100
	gravityY := -ay * 100

	if ax == 0 && ay == 0 {
		gravityY = physics.DefaultGravityY
	}

	g.physicsManager.SetGravity(gravityX, gravityY)
	g.drawer.HandleMouseEvent(g.physicsManager.GetSpace())
	g.physicsManager.Step(1 / 60.0)
}

func (g *Game) handleInput() {
	if clicked, x, y := g.inputHandler.CheckMouseClick(); clicked {
		g.handleButtonClick(x, y)
	}

	for _, touch := range g.inputHandler.CheckTouchInput() {
		g.handleButtonClick(touch.X, touch.Y)
	}
}

func (g *Game) handleButtonClick(x, y int) {
	if g.inputHandler.IsButtonClicked(x, y, ui.SpeakerButtonConfig) {
		g.state.SetMuted(!g.state.IsMuted())
		sound.SetMuted(g.state.IsMuted())
	}

	if g.state.ShowGameOverDialog && g.inputHandler.IsRetryButtonClicked(x, y) {
		g.resetGame()
	}
}

func (g *Game) checkGameOver() {
	if g.state.ShowGameOverDialog {
		return
	}

	if g.physicsManager.CheckBodiesOutOfBounds() {
		g.state.TriggerGameOver()
		if g.state.GameOverSE {
			sound.PlayGameOver()
			sound.StopBackgroundMusic()
		}
	}

	if g.state.GameOver && !g.state.ShowGameOverDialog {
		g.state.PrepareGameOverDialog()
		g.physicsManager.StopAllBodies()
	}
}

func (g *Game) updateDropLogic() {
	if g.state.ShowGameOverDialog {
		return
	}

	g.state.IncrementDropCount()

	if g.state.DropCount >= 45 {
		g.dropFruit()
		g.state.ResetDropCount()
	}
}

func (g *Game) updateAnimation() {
	g.state.NextFruit.Angle += 0.01
}

func (g *Game) dropFruit() {
	if !g.physicsManager.CanSpawnAt(g.state.NextFruit.X, g.state.NextFruit.Y, physics.SpawnCheckRadius) {
		g.state.SpawnFailCount++
		if g.state.SpawnFailCount >= physics.MaxSpawnFailures {
			g.state.TriggerGameOver()
			if g.state.GameOverSE {
				sound.PlayGameOver()
				sound.StopBackgroundMusic()
			}
		}
		return
	}

	g.state.SpawnFailCount = 0

	addData := physics.AddShapeData{
		Kind:  g.state.NextFruit.Kind,
		Pos:   cp.Vector{X: g.state.NextFruit.X, Y: g.state.NextFruit.Y},
		Angle: g.state.NextFruit.Angle,
	}
	g.physicsManager.GetSpace().AddPostStepCallback(
		physics.CreateAddShapeCallback(g.physicsManager),
		g.state.NextFruit.Kind,
		addData,
	)

	g.state.NextFruit.Kind = assets.Kind(rand.Intn(2) + int(assets.Min))
	g.state.NextFruit.X = float64(rand.Intn(ui.ScreenWidth-100) + 50)
	g.state.NextFruit.Y = float64(rand.Intn(ui.ScreenHeight-100) + 50)
	g.state.NextFruit.Angle = rand.Float64() * 2 * math.Pi
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.state.ShowTitleScreen {
		g.renderer.DrawTitleScreen(screen, physics.PaddingBottom)
		return
	}

	g.renderer.DrawBackground(screen, physics.PaddingBottom)

	g.physicsManager.GetSpace().EachShape(func(shape *cp.Shape) {
		if polyShape, ok := shape.Class.(*cp.PolyShape); ok {
			vec := polyShape.Body().Position()
			kind := polyShape.Body().UserData.(assets.Kind)
			g.renderer.DrawFruit(screen, kind, vec.X, vec.Y-physics.PaddingBottom, polyShape.Body().Angle())
		}
	})

	if g.debug {
		cp.DrawSpace(g.physicsManager.GetSpace(), g.drawer.WithScreen(screen))
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf(
		"FPS: %0.2f  The Go gopher was designed by Renee French.\nScore: %d\nHiScore: %d",
		ebiten.ActualFPS(),
		g.state.Score,
		g.state.HiScore,
	))

	g.renderer.DrawSpeakerButton(screen, g.state.IsMuted())

	if g.state.ShowGameOverDialog {
		g.renderer.DrawGameOverDialog(screen, g.state.FinalScore, g.state.FinalWatermelonHits)

		if g.state.GameOverScreenshot == nil {
			g.state.GameOverScreenshot = ebiten.NewImage(ui.ScreenWidth, ui.ScreenHeight)
			g.state.GameOverScreenshot.DrawImage(screen, nil)
			shareGameResultToX(g.state.GameOverScreenshot, g.state.FinalScore, g.state.FinalWatermelonHits)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ui.ScreenWidth, ui.ScreenHeight
}

func (g *Game) resetGame() {
	g.physicsManager.ScheduleRemoveAllFruits()
	g.state.Reset()
	hideShareButton()

	if !g.state.IsMuted() {
		sound.StartBackgroundMusic()
	}
}

func createCollisionHandler(physManager *physics.Manager) func(*cp.Arbiter, *cp.Space, interface{}) bool {
	return func(arb *cp.Arbiter, space *cp.Space, data interface{}) bool {
		shape1, shape2 := arb.Shapes()

		kind1, ok1 := shape1.Body().UserData.(assets.Kind)
		kind2, ok2 := shape2.Body().UserData.(assets.Kind)
		if !ok1 || !ok2 {
			return false
		}

		if kind1 == assets.Watermelon && kind2 == assets.Watermelon {
			currentGame.state.IncrementWatermelonHits()
		}

		space.AddPostStepCallback(physics.CreateRemoveShapeCallback(physManager), shape1, nil)
		space.AddPostStepCallback(physics.CreateRemoveShapeCallback(physManager), shape2, nil)

		currentGame.state.AddScore(kind1.Score())

		if kind1 == assets.Melon || kind1 == assets.Watermelon {
			sound.PlaySuikaJoin()
		} else {
			sound.PlayJoin()
		}

		hasNext, nextKind := kind1.Next()
		if !hasNext {
			return false
		}

		pos := shape1.Body().Position().Clone()
		pos.Sub(shape2.Body().Position()).Mult(0.5).Add(shape2.Body().Position())
		angle := (shape1.Body().Angle() + shape2.Body().Angle()) / 2

		addData := physics.AddShapeData{
			Kind:  nextKind,
			Pos:   pos,
			Angle: angle,
		}
		space.AddPostStepCallback(physics.CreateAddShapeCallback(physManager), nextKind, addData)

		return false
	}
}

func main() {
	setupWASMCallbacks()

	game := NewGame()
	currentGame = game

	ebiten.SetWindowSize(ui.ScreenWidth, ui.ScreenHeight)
	ebiten.SetWindowTitle("Suika Shaker")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
