package main

import (
	"io"
	"strings"
	"testing"
)

func TestRunValidate_OK(t *testing.T) {
	tmp := t.TempDir()
	m := writeFile(t, tmp, "manifest.yaml", validManifestYAML)
	if err := runValidate([]string{m}, io.Discard); err != nil {
		t.Fatalf("runValidate: unexpected error: %v", err)
	}
}

func TestRunValidate_RejectsACL(t *testing.T) {
	tmp := t.TempDir()
	m := writeFile(t, tmp, "manifest.yaml", validManifestYAML+"acl:\n  exposure_scope: instance\n")
	err := runValidate([]string{m}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "acl") {
		t.Fatalf("expected acl error, got: %v", err)
	}
}

func TestRunValidate_RejectsBrokenManifest(t *testing.T) {
	tmp := t.TempDir()
	// Пустой bundle → manifest.Validate должен вернуть ошибку.
	broken := strings.Replace(validManifestYAML, "bundle: demo", "bundle: \"\"", 1)
	m := writeFile(t, tmp, "manifest.yaml", broken)
	if err := runValidate([]string{m}, io.Discard); err == nil {
		t.Fatal("expected validation error for empty bundle, got nil")
	}
}
