package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runPack валидирует манифест и собирает плоский source.zip для загрузки:
// без *_test.go и подкаталогов, с go.mod (переименовывая go.mod.txt).
// usage: skillgen pack <dir>
func runPack(args []string, stdout io.Writer) error {
	dir := "."
	if len(args) > 0 && args[0] != "" {
		dir = args[0]
	}
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := runValidate([]string{manifestPath}, io.Discard); err != nil {
		return fmt.Errorf("manifest check failed: %w", err)
	}

	src := filepath.Join(dir, "source")
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read source dir: %w", err)
	}

	// Собрать плоский список (имя в zip -> путь на диске), отфильтровав тесты,
	// переименовав go.mod.txt и отвергнув подкаталоги.
	picked := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			return fmt.Errorf("source/ must be flat: unexpected subdirectory %q", name)
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		zipName := name
		if name == "go.mod.txt" {
			zipName = "go.mod"
		}
		picked[zipName] = filepath.Join(src, name)
	}

	for _, required := range []string{"main.go", "mcphub_gen.go", "go.mod"} {
		if _, ok := picked[required]; !ok {
			return fmt.Errorf("missing %s in source/ (run skillgen gen / init first)", required)
		}
	}

	outPath := filepath.Join(dir, "source.zip")
	if err := writeFlatZip(outPath, picked); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote %s — upload manifest.yaml + source.zip on the «Загрузить готовое» tab\n", outPath)
	return nil
}

func writeFlatZip(outPath string, files map[string]string) error {
	zf, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)
	for zipName, diskPath := range files {
		w, err := zw.Create(zipName)
		if err != nil {
			return fmt.Errorf("zip entry %s: %w", zipName, err)
		}
		data, err := os.ReadFile(diskPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", diskPath, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write %s: %w", zipName, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zip: %w", err)
	}
	return nil
}
