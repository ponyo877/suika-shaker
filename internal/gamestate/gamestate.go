package gamestate

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	assets "github.com/ponyo877/suika-shaker/assets/image"
)

type State struct {
	Count               int
	DropCount           int
	SpawnFailCount      int
	Score               int
	HiScore             int
	WatermelonHits      int
	NextFruit           NextFruit
	GameOver            bool
	GameOverSE          bool
	Muted               bool
	ShowGameOverDialog  bool
	ShowTitleScreen     bool
	GameOverScreenshot  *ebiten.Image
	FinalScore          int
	FinalWatermelonHits int
}

type NextFruit struct {
	Kind  assets.Kind
	X     float64
	Y     float64
	Angle float64
}

func NewState() *State {
	return &State{
		Score:           0,
		HiScore:         0,
		WatermelonHits:  0,
		ShowTitleScreen: true,
	}
}

func (s *State) IncrementCount() {
	s.Count++
}

func (s *State) IncrementDropCount() {
	s.DropCount++
}

func (s *State) ResetDropCount() {
	s.DropCount = 0
}

func (s *State) AddScore(points int) {
	s.Score += points
}

func (s *State) IncrementWatermelonHits() {
	s.WatermelonHits++
}

func (s *State) TriggerGameOver() {
	if !s.GameOver {
		s.GameOver = true
	}
	if !s.GameOverSE {
		s.GameOverSE = true
	}
}

func (s *State) PrepareGameOverDialog() {
	s.ShowGameOverDialog = true
	s.FinalScore = s.Score
	s.FinalWatermelonHits = s.WatermelonHits
	s.HiScore = int(math.Max(float64(s.Score), float64(s.HiScore)))
}

func (s *State) Reset() {
	s.Score = 0
	s.WatermelonHits = 0
	s.GameOver = false
	s.GameOverSE = false
	s.ShowGameOverDialog = false
	s.GameOverScreenshot = nil
	s.FinalScore = 0
	s.FinalWatermelonHits = 0
	s.SpawnFailCount = 0
}

func (s *State) SetMuted(muted bool) {
	s.Muted = muted
}

func (s *State) IsMuted() bool {
	return s.Muted
}
