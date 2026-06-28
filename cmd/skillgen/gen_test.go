package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// gen должен сам создать несуществующий --out каталог.
func TestRunGen_CreatesOutDir(t *testing.T) {
	tmp := t.TempDir()
	mPath := writeFile(t, tmp, "manifest.yaml", validManifestYAML)
	out := filepath.Join(tmp, "nested", "source")

	if err := runGen([]string{"--manifest", mPath, "--out", out}, io.Discard); err != nil {
		t.Fatalf("runGen: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "mcphub_gen.go")); err != nil {
		t.Fatalf("mcphub_gen.go not written: %v", err)
	}
}
