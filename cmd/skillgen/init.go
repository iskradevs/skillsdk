package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runInit создаёт каркас навыка: manifest.yaml + source/{main.go, skill.go,
// skill_test.go, go.mod} и сразу генерирует mcphub_gen.go.
// usage: skillgen init <bundle> [parentDir]
func runInit(args []string, stdout io.Writer) (err error) {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("usage: skillgen init <bundle> [dir]")
	}
	bundle := args[0]
	parent := "."
	if len(args) > 1 {
		parent = args[1]
	}
	root := filepath.Join(parent, bundle)
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		return fmt.Errorf("target %s already exists and is not empty", root)
	}
	toolID := strings.ReplaceAll(bundle, "-", "_")
	src := filepath.Join(root, "source")
	if err := os.MkdirAll(src, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	// Откатываем частично созданный каркас при любой ошибке ниже: каталог был
	// пуст или отсутствовал по проверке выше, так что удаление безопасно.
	defer func() {
		if err != nil {
			os.RemoveAll(root)
		}
	}()

	files := map[string]string{
		filepath.Join(root, "manifest.yaml"): initManifest(bundle, toolID),
		filepath.Join(src, "main.go"):        initMainGo(toolID),
		filepath.Join(src, "skill.go"):       initSkillGo(),
		filepath.Join(src, "skill_test.go"):  initSkillTestGo(),
		filepath.Join(src, "go.mod"):         fmt.Sprintf("module mcphubguest/%s\n\ngo 1.26\n", bundle),
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	// Сгенерировать mcphub_gen.go из свежего манифеста.
	if err := runGen([]string{"--manifest", filepath.Join(root, "manifest.yaml"), "--out", src}, io.Discard); err != nil {
		return fmt.Errorf("generate wrappers: %w", err)
	}
	fmt.Fprintf(stdout, "scaffolded %s — edit manifest.yaml + source/main.go, then: skillgen validate manifest.yaml && skillgen pack .\n", root)
	return nil
}

func initManifest(bundle, toolID string) string {
	return fmt.Sprintf(`schema_version: 1
bundle: %s
version: "0.1.0"
author: ""
description: "TODO: что делает навык"
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
  - id: %s
    description: "TODO: что делает инструмент"
    input_schema:
      type: object
      properties:
        text: { type: string }
      required: [text]
    capabilities_required: []
`, bundle, toolID)
}

// В шаблонах ниже backtick (struct-теги) не может быть внутри Go raw-string,
// поэтому используем placeholder § → ` и заменяем в конце.
func initMainGo(toolID string) string {
	tmpl := `//go:build tinygo

package main

import (
	"encoding/json"
	"unsafe"
)

func main() {}

type okEnv struct {
	OK     bool            §json:"ok"§
	Result json.RawMessage §json:"result"§
}

type errEnv struct {
	OK    bool    §json:"ok"§
	Error errBody §json:"error"§
}

type errBody struct {
	Code    string §json:"code"§
	Message string §json:"message"§
}

//go:wasmexport handle
func handle(argsPtr, argsLen int32) int64 {
	input := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(uint32(argsPtr)))), argsLen)
	var in struct {
		Tool string          §json:"tool"§
		Args json.RawMessage §json:"args"§
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return errEnvelope("bad_input", "не удалось разобрать ввод")
	}
	if in.Tool != "TOOLID" {
		return errEnvelope("unknown_tool", "неизвестный инструмент")
	}
	var a struct {
		Text string §json:"text"§
	}
	_ = json.Unmarshal(in.Args, &a)
	return okEnvelope(transform(a.Text))
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
`
	tmpl = strings.ReplaceAll(tmpl, "TOOLID", toolID)
	return strings.ReplaceAll(tmpl, "§", "`")
}

// initSkillGo — чистая логика без build-тега: тестируется обычным go test.
func initSkillGo() string {
	tmpl := `package main

// toolResult — форма результата инструмента (контракт выхода).
type toolResult struct {
	Text string §json:"text"§
}

// transform — бизнес-логика инструмента. TODO: реализуй.
func transform(text string) toolResult {
	return toolResult{Text: text}
}
`
	return strings.ReplaceAll(tmpl, "§", "`")
}

func initSkillTestGo() string {
	return `package main

import "testing"

func TestTransform(t *testing.T) {
	if got := transform("привет"); got.Text != "привет" {
		t.Fatalf("transform: got %q, want %q", got.Text, "привет")
	}
}
`
}
