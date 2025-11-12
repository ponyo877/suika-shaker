package sound

import (
	"bytes"
	_ "embed"
	"io"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 48000

var (
	//go:embed background.wav
	backgroundWAV []byte
	//go:embed gameover.wav
	gameoverWAV []byte
	//go:embed join.wav
	joinWAV []byte
	//go:embed suikajoin.wav
	suikajoinWAV []byte
)

type Manager struct {
	context          *audio.Context
	backgroundPlayer *audio.Player
	gameoverData     []byte
	joinData         []byte
	suikajoinData    []byte
	muted            bool
}

var defaultManager *Manager

func init() {
	defaultManager = NewManager()
}

func NewManager() *Manager {
	ctx := audio.NewContext(sampleRate)

	backgroundStream, err := wav.DecodeF32(bytes.NewReader(backgroundWAV))
	if err != nil {
		log.Fatal(err)
	}

	loopStream := audio.NewInfiniteLoop(backgroundStream, backgroundStream.Length())
	bgPlayer, err := ctx.NewPlayerF32(loopStream)
	if err != nil {
		log.Fatal(err)
	}
	bgPlayer.SetVolume(0.3)

	return &Manager{
		context:          ctx,
		backgroundPlayer: bgPlayer,
		gameoverData:     decodeToBytes(gameoverWAV),
		joinData:         decodeToBytes(joinWAV),
		suikajoinData:    decodeToBytes(suikajoinWAV),
		muted:            false,
	}
}

func decodeToBytes(wavData []byte) []byte {
	stream, err := wav.DecodeF32(bytes.NewReader(wavData))
	if err != nil {
		log.Fatal(err)
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func (m *Manager) SetMuted(muted bool) {
	m.muted = muted
	if muted {
		m.StopBackgroundMusic()
	} else {
		m.StartBackgroundMusic()
	}
}

func (m *Manager) IsMuted() bool {
	return m.muted
}

func (m *Manager) PlayGameOver() {
	if m.muted {
		return
	}
	player := m.context.NewPlayerF32FromBytes(m.gameoverData)
	player.Play()
}

func (m *Manager) PlayJoin() {
	if m.muted {
		return
	}
	player := m.context.NewPlayerF32FromBytes(m.joinData)
	player.Play()
}

func (m *Manager) PlaySuikaJoin() {
	if m.muted {
		return
	}
	player := m.context.NewPlayerF32FromBytes(m.suikajoinData)
	player.Play()
}

func (m *Manager) StartBackgroundMusic() {
	if m.backgroundPlayer != nil && !m.backgroundPlayer.IsPlaying() {
		m.backgroundPlayer.Play()
	}
}

func (m *Manager) StopBackgroundMusic() {
	if m.backgroundPlayer != nil && m.backgroundPlayer.IsPlaying() {
		m.backgroundPlayer.Pause()
	}
}

func SetMuted(muted bool) {
	defaultManager.SetMuted(muted)
}

func PlayGameOver() {
	defaultManager.PlayGameOver()
}

func PlayJoin() {
	defaultManager.PlayJoin()
}

func PlaySuikaJoin() {
	defaultManager.PlaySuikaJoin()
}

func StartBackgroundMusic() {
	defaultManager.StartBackgroundMusic()
}

func StopBackgroundMusic() {
	defaultManager.StopBackgroundMusic()
}
