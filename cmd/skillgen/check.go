package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// lookTinyGo находит бинарь tinygo (переопределяется в тестах).
var lookTinyGo = func() (string, error) { return exec.LookPath("tinygo") }

// runCheck прогоняет frozen-сборку гостя под TinyGo. Если TinyGo не установлен —
// мягко скипает (сервер всё равно соберёт WASM).
// usage: skillgen check <dir>
func runCheck(args []string, stdout io.Writer) error {
	dir := "."
	if len(args) > 0 && args[0] != "" {
		dir = args[0]
	}
	bin, err := lookTinyGo()
	if err != nil {
		fmt.Fprintln(stdout, "TinyGo не установлен — пропускаю локальную сборку (WASM соберёт сервер). Для проверки поставь TinyGo 0.41.1.")
		return nil
	}
	tmp, err := os.MkdirTemp("", "skillgen-check-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	out := filepath.Join(tmp, "out.wasm")
	cmdArgs := append([]string{"build", "-o", out}, frozenTinyGoFlags...)
	cmdArgs = append(cmdArgs, ".")
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = filepath.Join(dir, "source")
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tinygo build failed: %w", err)
	}
	fmt.Fprintln(stdout, "tinygo build OK")
	return nil
}
