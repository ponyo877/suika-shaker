package sound

import (
	"bytes"
	_ "embed"
	"io"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	sampleRate = 48000
)

var (
	//go:embed background.wav
	background_wav []byte
	//go:embed gameover.wav
	gameover_wav []byte
	//go:embed join.wav
	join_wav []byte
	//go:embed suikajoin.wav
	suikajoin_wav []byte

	AudioContext     *audio.Context
	BackgroundPlayer *audio.Player
	gameoverData     []byte
	joinData         []byte
	suikajoinData    []byte

	Muted bool = false // Global mute state
)

func init() {
	AudioContext = audio.NewContext(sampleRate)

	// Load background music
	backgroundStream, err := wav.DecodeF32(bytes.NewReader(background_wav))
	if err != nil {
		log.Fatal(err)
	}
	// Create infinite loop for background music
	loopStream := audio.NewInfiniteLoop(backgroundStream, backgroundStream.Length())
	BackgroundPlayer, err = AudioContext.NewPlayerF32(loopStream)
	if err != nil {
		log.Fatal(err)
	}
	BackgroundPlayer.SetVolume(0.3) // Set lower volume for background music

	// Pre-decode sound effects into bytes for quick playback
	gameoverData = decodeToBytes(gameover_wav)
	joinData = decodeToBytes(join_wav)
	suikajoinData = decodeToBytes(suikajoin_wav)
}

// decodeToBytes decodes WAV data to PCM bytes
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

// SetMuted sets the global mute state
func SetMuted(muted bool) {
	Muted = muted
	if muted {
		StopBackgroundMusic()
	} else {
		StartBackgroundMusic()
	}
}

// PlayGameOver plays the game over sound
func PlayGameOver() {
	if Muted {
		return
	}
	player := AudioContext.NewPlayerF32FromBytes(gameoverData)
	player.Play()
}

// PlayJoin plays the join sound when fruits merge
func PlayJoin() {
	if Muted {
		return
	}
	player := AudioContext.NewPlayerF32FromBytes(joinData)
	player.Play()
}

// PlaySuikaJoin plays the special sound when melon or watermelon merge
func PlaySuikaJoin() {
	if Muted {
		return
	}
	player := AudioContext.NewPlayerF32FromBytes(suikajoinData)
	player.Play()
}

// StartBackgroundMusic starts playing the background music
func StartBackgroundMusic() {
	if BackgroundPlayer != nil && !BackgroundPlayer.IsPlaying() {
		BackgroundPlayer.Play()
	}
}

// StopBackgroundMusic stops the background music
func StopBackgroundMusic() {
	if BackgroundPlayer != nil && BackgroundPlayer.IsPlaying() {
		BackgroundPlayer.Pause()
	}
}
