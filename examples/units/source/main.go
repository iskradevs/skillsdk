//go:build tinygo

package main

import (
	"encoding/json"
	"unsafe"
)

func main() {}

type okEnv struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type errEnv struct {
	OK    bool    `json:"ok"`
	Error errBody `json:"error"`
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type convertResult struct {
	Value    float64 `json:"value"`
	From     string  `json:"from"`
	To       string  `json:"to"`
	Result   float64 `json:"result"`
	Category string  `json:"category"`
}

//go:wasmexport handle
func handle(argsPtr, argsLen int32) int64 {
	input := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(uint32(argsPtr)))), argsLen)
	var in struct {
		Tool string          `json:"tool"`
		Args json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return errEnvelope("bad_input", "не удалось разобрать ввод")
	}
	if in.Tool != "unit_convert" {
		return errEnvelope("unknown_tool", "неизвестный инструмент")
	}
	var a struct {
		Value float64 `json:"value"`
		From  string  `json:"from"`
		To    string  `json:"to"`
	}
	_ = json.Unmarshal(in.Args, &a)
	res, cat, ok := convertUnits(a.Value, a.From, a.To)
	if !ok {
		return errEnvelope("bad_units", "неизвестные или несовместимые единицы")
	}
	return okEnvelope(convertResult{Value: a.Value, From: a.From, To: a.To, Result: res, Category: cat})
}

func okEnvelope(result any) int64 {
	rb, err := json.Marshal(result)
	if err != nil {
		return errEnvelope("marshal", "не удалось сериализовать результат")
	}
	b, _ := json.Marshal(okEnv{OK: true, Result: rb})
	return writeJSON(b)
}

func errEnvelope(code, message string) int64 {
	b, _ := json.Marshal(errEnv{OK: false, Error: errBody{Code: code, Message: message}})
	return writeJSON(b)
}
