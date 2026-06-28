package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/iskradevs/skillsdk/manifest"
)

// runValidate проверяет манифест структурно (manifest.Validate), запрещает
// author-acl и (best-effort) сверяет локальную версию TinyGo с toolchain.
func runValidate(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillgen validate <manifest.yaml>")
	}
	m, raw, err := loadManifest(args[0])
	if err != nil {
		return err
	}
	if manifestHasACL(raw) {
		return fmt.Errorf("manifest has an acl block: authors must not set acl; the platform assigns scope on upload (remove it)")
	}
	// Платформа проставляет acl.exposure_scope при ingest ДО manifest.Validate,
	// а validateACL вызывается безусловно. Поэтому локально подставляем валидный
	// stub, чтобы прогнать структурную проверку остального — author-acl уже
	// отвергнут выше.
	m.ACL = manifest.ACL{ExposureScope: "instance"}
	if err := manifest.Validate(&m); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}
	warnToolchainMismatch(m.Source.Toolchain, stdout)
	fmt.Fprintln(stdout, "manifest OK")
	return nil
}

// warnToolchainMismatch печатает предупреждение, если локальный TinyGo не
// совпадает с toolchain из манифеста. Не fatal: сервер — источник истины сборки.
func warnToolchainMismatch(toolchain string, stdout io.Writer) {
	if toolchain == "" {
		return
	}
	bin, err := exec.LookPath("tinygo")
	if err != nil {
		fmt.Fprintf(stdout, "note: TinyGo не найден локально; манифест требует %s (соберёт сервер)\n", toolchain)
		return
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return
	}
	// Манифест: "tinygo-0.41.1" → версия "0.41.1".
	want := strings.TrimPrefix(toolchain, "tinygo-")
	if want != "" && !strings.Contains(string(out), want) {
		fmt.Fprintf(stdout, "warning: локальный TinyGo (%s) не совпадает с toolchain манифеста %s\n",
			strings.TrimSpace(string(out)), toolchain)
	}
}
