//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/bonsai/minyo-go/internal/geo"
	"github.com/bonsai/minyo-go/internal/game"
)

func main() {
	api := js.Global().Get("Object").New()
	api.Set("version", "0.1.0")
	api.Set("distanceKm", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 4 {
			return js.Null()
		}
		lat1, lon1 := args[0].Float(), args[1].Float()
		lat2, lon2 := args[2].Float(), args[3].Float()
		if !geo.ValidCoordinates(lat1, lon1) || !geo.ValidCoordinates(lat2, lon2) {
			return js.Null()
		}
		return geo.DistanceKm(lat1, lon1, lat2, lon2)
	}))
	api.Set("validCoordinates", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 {
			return false
		}
		return geo.ValidCoordinates(args[0].Float(), args[1].Float())
	}))
	api.Set("validateAnswerIDs", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 || args[0].Type() != js.TypeObject || args[1].Type() != js.TypeObject {
			return false
		}
		expected := make([]int64, args[0].Length())
		submitted := make([]int64, args[1].Length())
		for i := range expected { expected[i] = int64(args[0].Index(i).Int()) }
		for i := range submitted { submitted[i] = int64(args[1].Index(i).Int()) }
		return game.ValidateAnswerIDs(expected, submitted)
	}))
	api.Set("passed", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 { return false }
		return game.Passed(args[0].Int(), args[1].Int())
	}))
	js.Global().Set("MinyoGo", api)

	// Keep exported callbacks available for the lifetime of the page.
	select {}
}
