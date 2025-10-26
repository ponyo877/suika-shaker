//go:build !js || !wasm

package main

func setupWASMCallbacks() {
	// No-op for native builds
}

func getAcceleration() (float64, float64, float64) {
	// Return zero acceleration for native builds
	return 0, 0, 0
}
