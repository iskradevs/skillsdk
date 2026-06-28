package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunInit_Scaffold(t *testing.T) {
	tmp := t.TempDir()
	if err := runInit([]string{"demo", tmp}, io.Discard); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	for _, f := range []string{
		"manifest.yaml",
		"source/main.go",
		"source/skill.go",
		"source/skill_test.go",
		"source/go.mod",
		"source/mcphub_gen.go",
	} {
		if _, err := os.Stat(filepath.Join(tmp, "demo", f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	// Манифест без acl и проходит author-валидацию (runValidate подставляет
	// acl-stub внутри, как платформенный ingest).
	mPath := filepath.Join(tmp, "demo", "manifest.yaml")
	raw, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifestHasACL(raw) {
		t.Error("scaffolded manifest must not contain acl")
	}
	if err := runValidate([]string{mPath}, io.Discard); err != nil {
		t.Errorf("scaffolded manifest failed validate: %v", err)
	}
}

func TestRunInit_RefusesNonEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "demo"), "existing.txt", "x")
	if err := runInit([]string{"demo", tmp}, io.Discard); err == nil {
		t.Fatal("expected refusal on non-empty target dir")
	}
}
