package main

import "testing"

func TestPctEncode(t *testing.T) {
	if pctEncode("Saint Petersburg") != "Saint%20Petersburg" {
		t.Fatalf("pctEncode пробел")
	}
	if pctEncode("Мир") == "Мир" {
		t.Fatalf("кириллица должна кодироваться")
	}
}

func TestFtoa(t *testing.T) {
	if ftoa(55.7558) != "55.7558" {
		t.Fatalf("ftoa positive: %q", ftoa(55.7558))
	}
	if ftoa(-0.5) != "-0.5000" {
		t.Fatalf("ftoa negative: %q", ftoa(-0.5))
	}
}

func TestWmo(t *testing.T) {
	if wmoText(0) != "ясно" || wmoText(95) != "гроза" {
		t.Fatalf("wmoText known")
	}
	if wmoText(999) != "неизвестно" {
		t.Fatalf("wmoText unknown")
	}
}
