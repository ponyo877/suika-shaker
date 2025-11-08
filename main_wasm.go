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
	fmt.Println("shareGameResultToX called with score:", score, "watermelonHits:", watermelonHits)

	if screenshot == nil {
		fmt.Println("Error: screenshot is nil")
		return
	}

	// Encode screenshot to PNG
	fmt.Println("Encoding screenshot to PNG...")
	var buf bytes.Buffer
	if err := png.Encode(&buf, screenshot); err != nil {
		fmt.Println("Failed to encode screenshot:", err)
		return
	}
	fmt.Println("Screenshot encoded, size:", buf.Len(), "bytes")

	// Convert to base64
	fmt.Println("Converting to base64...")
	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Println("Base64 conversion complete, length:", len(base64Image))

	// Create share text
	url := "https://ponyo877.github.io/suika-shaker/"
	shareText := fmt.Sprintf("Suika Shaker\nScore: %d\nWatermelon Hits: %d\n%s", score, watermelonHits, url)
	fmt.Println("Share text:", shareText)

	// Show share button with screenshot data
	// The HTML button will handle the actual sharing in user gesture context
	fmt.Println("Calling JavaScript showShareButton function...")
	if js.Global().Get("showShareButton").Truthy() {
		js.Global().Call("showShareButton", base64Image, shareText)
		fmt.Println("JavaScript showShareButton function called")
	} else {
		fmt.Println("Warning: showShareButton not available")
	}
}

func hideShareButton() {
	fmt.Println("hideShareButton called")
	if js.Global().Get("hideShareButton").Truthy() {
		js.Global().Call("hideShareButton")
		fmt.Println("JavaScript hideShareButton function called")
	} else {
		fmt.Println("Warning: hideShareButton not available")
	}
}
