package main

import (
	"errors"
	"io"
	"testing"
)

// Если TinyGo не найден — check мягко скипает (exit 0), не запуская сборку.
func TestRunCheck_SkipsWithoutTinyGo(t *testing.T) {
	orig := lookTinyGo
	lookTinyGo = func() (string, error) { return "", errors.New("not found") }
	defer func() { lookTinyGo = orig }()

	if err := runCheck([]string{t.TempDir()}, io.Discard); err != nil {
		t.Fatalf("runCheck should skip gracefully, got: %v", err)
	}
}
