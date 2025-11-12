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

var (
	accelerationX float64
	accelerationY float64
	accelerationZ float64
)

func setupWASMCallbacks() {
	js.Global().Set("setAcceleration", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 3 {
			accelerationX = args[0].Float()
			accelerationY = args[1].Float()
			accelerationZ = args[2].Float()
		}
		return nil
	}))

	js.Global().Set("startGameFromJS", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if currentGame != nil {
			currentGame.showTitleScreen = false

			if js.Global().Get("onGameStarted").Truthy() {
				js.Global().Call("onGameStarted")
			}
		}
		return nil
	}))

	js.Global().Set("startAudioContext", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if currentGame != nil && !currentGame.muted {
			sound.StartBackgroundMusic()
		}
		return nil
	}))
}

func getAcceleration() (float64, float64, float64) {
	return accelerationX, accelerationY, accelerationZ
}

func shareGameResultToX(screenshot *ebiten.Image, score int, watermelonHits int) {
	if screenshot == nil {
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, screenshot); err != nil {
		return
	}

	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())

	url := "https://ponyo877.github.io/suika-shaker/"
	shareText := fmt.Sprintf("Suika Shaker\nScore: %d\nWatermelon Hits: %d\n%s", score, watermelonHits, url)

	if js.Global().Get("showShareButton").Truthy() {
		js.Global().Call("showShareButton", base64Image, shareText)
	}
}

func hideShareButton() {
	if js.Global().Get("hideShareButton").Truthy() {
		js.Global().Call("hideShareButton")
	}
}
