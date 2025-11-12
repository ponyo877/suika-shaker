package physics

import (
	"github.com/jakecoffman/cp/v2"
	assets "github.com/ponyo877/suika-shaker/assets/image"
	"github.com/ponyo877/suika-shaker/internal/ui"
)

const (
	SpawnCheckRadius  = 40
	MaxSpawnFailures  = 3
	ContainerHeight   = 800
	PaddingBottom     = 0
	WallThickness     = 1
	WallElasticity    = 0.6
	WallFriction      = 0.4
	FruitElasticity   = 0.2
	FruitFriction     = 0.9
	FruitMassFactor   = 0.001
	SpaceIterations   = 30
	SleepTimeThreshold = 0.5
	DefaultGravityY   = 500
)

type Manager struct {
	space *cp.Space
}

func NewManager() *Manager {
	space := cp.NewSpace()
	space.Iterations = SpaceIterations
	space.SetGravity(cp.Vector{X: 0, Y: DefaultGravityY})
	space.SleepTimeThreshold = SleepTimeThreshold
	space.SetDamping(1)

	walls := []cp.Vector{
		{X: 0, Y: 0}, {X: 0, Y: ui.ScreenHeight},
		{X: ui.ScreenWidth, Y: 0}, {X: ui.ScreenWidth, Y: ui.ScreenHeight},
		{X: 0, Y: ui.ScreenHeight}, {X: ui.ScreenWidth, Y: ui.ScreenHeight},
		{X: 0, Y: 0}, {X: ui.ScreenWidth, Y: 0},
	}

	for i := 0; i < len(walls)-1; i += 2 {
		shape := space.AddShape(cp.NewSegment(space.StaticBody, walls[i], walls[i+1], WallThickness))
		shape.SetElasticity(WallElasticity)
		shape.SetFriction(WallFriction)
	}

	return &Manager{space: space}
}

func (m *Manager) GetSpace() *cp.Space {
	return m.space
}

func (m *Manager) SetGravity(x, y float64) {
	m.space.SetGravity(cp.Vector{X: x, Y: y})
}

func (m *Manager) Step(dt float64) {
	m.space.Step(dt)
}

func (m *Manager) AddFruit(kind assets.Kind, position cp.Vector, angle float64) {
	if !assets.Exists(kind) {
		return
	}
	imgSet := assets.Get(kind)

	body := m.space.AddBody(cp.NewBody(0, cp.MomentForPoly(10, len(imgSet.Vectors), imgSet.Vectors, cp.Vector{}, 1)))
	body.SetPosition(position)
	body.SetAngle(angle)
	body.UserData = kind

	fruit := m.space.AddShape(cp.NewPolyShape(body, len(imgSet.Vectors), imgSet.Vectors, cp.NewTransformIdentity(), 0))
	body.SetMass(fruit.Area() * FruitMassFactor)
	fruit.SetElasticity(FruitElasticity)
	fruit.SetFriction(FruitFriction)
	fruit.SetCollisionType(cp.CollisionType(kind))

	body.Activate()
	m.space.ReindexShape(fruit)
}

func (m *Manager) RemoveFruit(shape *cp.Shape) {
	m.space.RemoveBody(shape.Body())
	m.space.RemoveShape(shape)
}

func (m *Manager) CanSpawnAt(x, y, radius float64) bool {
	info := m.space.PointQueryNearest(cp.Vector{X: x, Y: y}, radius, cp.ShapeFilter{})
	if info.Shape == nil {
		return true
	}
	return info.Shape.Body().UserData == nil
}

func (m *Manager) StopAllBodies() {
	m.space.EachBody(func(body *cp.Body) {
		if body.UserData != nil {
			body.SetVelocity(0, 0)
			body.SetAngularVelocity(0)
		}
	})
}

func (m *Manager) ScheduleRemoveAllFruits() {
	m.space.EachShape(func(shape *cp.Shape) {
		if shape.Body().UserData != nil {
			m.space.AddPostStepCallback(
				CreateRemoveShapeCallback(m),
				shape,
				nil,
			)
		}
	})
}

func (m *Manager) CheckBodiesOutOfBounds() bool {
	outOfBounds := false
	m.space.EachBody(func(body *cp.Body) {
		x, y := body.Position().X, body.Position().Y
		if x < 0 || x > ui.ScreenWidth || y < 0 || y > ui.ScreenHeight {
			outOfBounds = true
		}
	})
	return outOfBounds
}

type AddShapeData struct {
	Kind  assets.Kind
	Pos   cp.Vector
	Angle float64
}

func CreateAddShapeCallback(manager *Manager) func(*cp.Space, interface{}, interface{}) {
	return func(space *cp.Space, key interface{}, data interface{}) {
		opt, ok := data.(AddShapeData)
		if !ok {
			return
		}
		manager.AddFruit(opt.Kind, opt.Pos, opt.Angle)
	}
}

func CreateRemoveShapeCallback(manager *Manager) func(*cp.Space, interface{}, interface{}) {
	return func(space *cp.Space, key interface{}, data interface{}) {
		shape, ok := key.(*cp.Shape)
		if !ok {
			return
		}
		manager.RemoveFruit(shape)
	}
}
