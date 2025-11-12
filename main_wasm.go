//go:build js && wasm

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ponyo877/suika-shaker/assets/sound"
)

type AccelerationData struct {
	X, Y, Z float64
}

var accelData AccelerationData

func setupWASMCallbacks() {
	js.Global().Set("setAcceleration", js.FuncOf(setAccelerationCallback))
	js.Global().Set("startGameFromJS", js.FuncOf(startGameCallback))
	js.Global().Set("startAudioContext", js.FuncOf(startAudioCallback))
}

func setAccelerationCallback(this js.Value, args []js.Value) interface{} {
	if len(args) >= 3 {
		accelData.X = args[0].Float()
		accelData.Y = args[1].Float()
		accelData.Z = args[2].Float()
	}
	return nil
}

func startGameCallback(this js.Value, args []js.Value) interface{} {
	if currentGame != nil {
		currentGame.state.ShowTitleScreen = false
		if js.Global().Get("onGameStarted").Truthy() {
			js.Global().Call("onGameStarted")
		}
	}
	return nil
}

func startAudioCallback(this js.Value, args []js.Value) interface{} {
	if currentGame != nil && !currentGame.state.IsMuted() {
		sound.StartBackgroundMusic()
	}
	return nil
}

func getAcceleration() (float64, float64, float64) {
	return accelData.X, accelData.Y, accelData.Z
}

func shareGameResultToX(screenshot *ebiten.Image, score, watermelonHits int) {
	if screenshot == nil {
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, screenshot); err != nil {
		return
	}

	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())
	shareText := fmt.Sprintf(
		"Suika Shaker\nScore: %d\nWatermelon Hits: %d\n%s",
		score,
		watermelonHits,
		"https://ponyo877.github.io/suika-shaker/",
	)

	if js.Global().Get("showShareButton").Truthy() {
		js.Global().Call("showShareButton", base64Image, shareText)
	}
}

func hideShareButton() {
	if js.Global().Get("hideShareButton").Truthy() {
		js.Global().Call("hideShareButton")
	}
}
