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
	// Expose setAcceleration function to JavaScript
	js.Global().Set("setAcceleration", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 3 {
			accelerationX = args[0].Float()
			accelerationY = args[1].Float()
			accelerationZ = args[2].Float()
		}
		return nil
	}))

	// Expose startGameFromJS function to JavaScript
	// Called when JavaScript handles START button tap in user gesture context
	js.Global().Set("startGameFromJS", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if currentGame != nil {
			currentGame.showTitleScreen = false
			// Audio will be started by JavaScript in user gesture context

			// Notify JavaScript that game has started (to hide START button)
			if js.Global().Get("onGameStarted").Truthy() {
				js.Global().Call("onGameStarted")
			}
		}
		return nil
	}))

	// Expose startAudioContext function to JavaScript
	// Called to ensure audio context starts in user gesture context
	js.Global().Set("startAudioContext", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Start background music if not muted
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

	// Encode screenshot to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, screenshot); err != nil {
		fmt.Println("Failed to encode screenshot:", err)
		return
	}

	// Convert to base64
	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Create share text
	shareText := fmt.Sprintf("Suika Shaker - Score: %d, Watermelon Hits: %d", score, watermelonHits)

	// Call JavaScript function to handle sharing
	js.Global().Call("shareToX", base64Image, shareText)
}
