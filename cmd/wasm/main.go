//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/primissus/ahorita/expr"
)

func main() {
	js.Global().Set("ahoritaRun", js.FuncOf(ahoritaRun))
	select {}
}

func ahoritaRun(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return marshalArgError("ahoritaRun requires input string")
	}
	input := args[0].String()

	optsJSON := "{}"
	if len(args) >= 2 {
		switch args[1].Type() {
		case js.TypeUndefined, js.TypeNull:
		case js.TypeString:
			optsJSON = args[1].String()
		default:
			return marshalInputError(input, "ahoritaRun optionsJSON must be a JSON string")
		}
	}

	return runBridge(input, optsJSON, runtimeEnv{
		now:       time.Now,
		defaultTZ: resolveBrowserTZ,
	})
}

func resolveBrowserTZ() string {
	intl := js.Global().Get("Intl")
	if !intl.Truthy() {
		return ""
	}
	dtf := intl.Call("DateTimeFormat")
	if !dtf.Truthy() {
		return ""
	}
	opts := dtf.Call("resolvedOptions")
	if !opts.Truthy() {
		return ""
	}
	tz := opts.Get("timeZone")
	if tz.Type() != js.TypeString {
		return ""
	}
	return tz.String()
}

func marshalArgError(msg string) string {
	b, _ := json.Marshal(expr.Result{
		Error: &expr.Error{Pos: -1, Msg: msg},
	})
	return string(b)
}

func marshalInputError(input, msg string) string {
	b, _ := json.Marshal(expr.Result{
		Input: input,
		Error: &expr.Error{Pos: -1, Msg: msg},
	})
	return string(b)
}
