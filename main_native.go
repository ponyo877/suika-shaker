//go:build !js || !wasm

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func setupWASMCallbacks() {
	// No-op for native builds
}

func getAcceleration() (float64, float64, float64) {
	// Return zero acceleration for native builds
	return 0, 0, 0
}

func shareGameResultToX(screenshot *ebiten.Image, score int, watermelonHits int) {
	// No-op for native builds
	// Could potentially save screenshot to file or open system share dialog
}

func hideShareButton() {
	// No-op for native builds
}
