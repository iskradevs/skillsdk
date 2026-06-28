package main

import (
	"os"
	"path/filepath"
	"testing"
)

// validManifestYAML — минимальный манифест, проходящий manifest.Validate,
// БЕЗ блока acl (author-style).
const validManifestYAML = `schema_version: 1
bundle: demo
version: "0.1.0"
description: "demo skill"
license: MIT
source:
  language: go-tinygo
  toolchain: tinygo-0.41.1
runtime:
  kind: wasm-per-call
  quotas:
    timeout_s: 10
    memory_mb: 32
    cpu_ms: 1000
tools:
  - id: demo
    description: "echo"
    input_schema:
      type: object
      properties:
        text: { type: string }
      required: [text]
    capabilities_required: []
`

// writeFile пишет содержимое в dir/name, создавая каталоги, и возвращает путь.
func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}
