package main

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestConvertLinear(t *testing.T) {
	r, cat, ok := convertUnits(1, "km", "m")
	if !ok || cat != "length" || !approx(r, 1000) {
		t.Fatalf("1 km != 1000 m: %v %s %v", r, cat, ok)
	}
	r, _, ok = convertUnits(2, "kg", "g")
	if !ok || !approx(r, 2000) {
		t.Fatalf("2 kg != 2000 g: %v", r)
	}
	r, _, ok = convertUnits(1024, "B", "KiB")
	if !ok || !approx(r, 1) {
		t.Fatalf("1024 B != 1 KiB: %v", r)
	}
}

func TestConvertTemp(t *testing.T) {
	r, cat, ok := convertUnits(100, "c", "f")
	if !ok || cat != "temperature" || !approx(r, 212) {
		t.Fatalf("100c != 212f: %v %s %v", r, cat, ok)
	}
	r, _, _ = convertUnits(0, "c", "k")
	if !approx(r, 273.15) {
		t.Fatalf("0c != 273.15k: %v", r)
	}
}

func TestConvertErrors(t *testing.T) {
	if _, _, ok := convertUnits(1, "km", "kg"); ok {
		t.Fatalf("разные категории должны давать ошибку")
	}
	if _, _, ok := convertUnits(1, "zzz", "m"); ok {
		t.Fatalf("неизвестная единица должна давать ошибку")
	}
}
