//go:build js && wasm

package main

import (
	"syscall/js"
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
}

func getAcceleration() (float64, float64, float64) {
	return accelerationX, accelerationY, accelerationZ
}
