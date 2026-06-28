package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/iskradevs/skillsdk/manifest"
)

// frozenTinyGoFlags — замороженные флаги сборки гостя (см. reference/abi.md).
// Единый источник для `skillgen check`.
var frozenTinyGoFlags = []string{
	"-target=wasm-unknown",
	"-buildmode=c-shared",
	"-scheduler=none",
	"-panic=trap",
}

// loadManifest читает и парсит manifest.yaml, возвращая также сырые байты
// (нужны для проверки наличия top-level acl).
func loadManifest(path string) (manifest.Manifest, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return manifest.Manifest{}, nil, fmt.Errorf("read manifest: %w", err)
	}
	var m manifest.Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return manifest.Manifest{}, nil, fmt.Errorf("parse manifest: %w", err)
	}
	return m, raw, nil
}

// manifestHasACL сообщает, присутствует ли top-level ключ acl в манифесте.
// Авторы не задают acl — scope проставляет платформа при ingest.
func manifestHasACL(raw []byte) bool {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal(raw, &top); err != nil {
		return false
	}
	_, ok := top["acl"]
	return ok
}
