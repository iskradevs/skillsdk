package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func setupBundle(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "manifest.yaml", validManifestYAML)
	writeFile(t, dir, "source/main.go", "//go:build tinygo\n\npackage main\n")
	writeFile(t, dir, "source/mcphub_gen.go", "//go:build tinygo\n\npackage main\n")
	writeFile(t, dir, "source/skill.go", "package main\n")
	writeFile(t, dir, "source/skill_test.go", "package main\n")
	writeFile(t, dir, "source/go.mod.txt", "module mcphubguest/demo\n\ngo 1.26\n")
	return dir
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func TestRunPack_FlatNoTestsRenamesGoMod(t *testing.T) {
	dir := setupBundle(t)
	if err := runPack([]string{dir}, io.Discard); err != nil {
		t.Fatalf("runPack: %v", err)
	}
	got := zipNames(t, filepath.Join(dir, "source.zip"))
	want := []string{"go.mod", "main.go", "mcphub_gen.go", "skill.go"}
	if len(got) != len(want) {
		t.Fatalf("zip entries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("zip entries = %v, want %v", got, want)
		}
	}
}

func TestRunPack_RejectsSubdir(t *testing.T) {
	dir := setupBundle(t)
	writeFile(t, dir, "source/sub/extra.go", "package main\n")
	if err := runPack([]string{dir}, io.Discard); err == nil {
		t.Fatal("expected error on nested subdir in source/")
	}
}

func TestRunPack_RequiresGenerated(t *testing.T) {
	dir := setupBundle(t)
	os.Remove(filepath.Join(dir, "source", "mcphub_gen.go"))
	if err := runPack([]string{dir}, io.Discard); err == nil {
		t.Fatal("expected error when mcphub_gen.go missing")
	}
}
