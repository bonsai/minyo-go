//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/bonsai/minyo-go/internal/geo"
	"github.com/bonsai/minyo-go/internal/game"
)

func isNumber(value js.Value) bool {
	return value.Type() == js.TypeNumber
}

func isInteger(value js.Value) bool {
	return isNumber(value) && js.Global().Get("Number").Call("isInteger", value).Bool()
}

func isNumberArray(value js.Value) bool {
	return value.Type() == js.TypeObject &&
		js.Global().Get("Array").Call("isArray", value).Bool()
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("version", "0.1.0")
	api.Set("distanceKm", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 4 {
			return js.Null()
		}
		for _, arg := range args {
			if !isNumber(arg) {
				return js.Null()
			}
		}
		lat1, lon1 := args[0].Float(), args[1].Float()
		lat2, lon2 := args[2].Float(), args[3].Float()
		if !geo.ValidCoordinates(lat1, lon1) || !geo.ValidCoordinates(lat2, lon2) {
			return js.Null()
		}
		return geo.DistanceKm(lat1, lon1, lat2, lon2)
	}))
	api.Set("validCoordinates", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 || !isNumber(args[0]) || !isNumber(args[1]) {
			return false
		}
		return geo.ValidCoordinates(args[0].Float(), args[1].Float())
	}))
	api.Set("validateAnswerIDs", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 || !isNumberArray(args[0]) || !isNumberArray(args[1]) ||
			args[0].Length() != 3 || args[1].Length() != 3 {
			return false
		}
		expected := make([]int64, 3)
		submitted := make([]int64, 3)
		for i := 0; i < 3; i++ {
			if !isInteger(args[0].Index(i)) || !isInteger(args[1].Index(i)) {
				return false
			}
			expected[i] = int64(args[0].Index(i).Int())
			submitted[i] = int64(args[1].Index(i).Int())
		}
		return game.ValidateAnswerIDs(expected, submitted)
	}))
	api.Set("passed", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 || !isInteger(args[0]) || !isInteger(args[1]) {
			return false
		}
		return game.Passed(args[0].Int(), args[1].Int())
	}))
	js.Global().Set("MinyoGo", api)

	// Keep exported callbacks available for the lifetime of the page.
	select {}
}
