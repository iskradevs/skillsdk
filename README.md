# Iskra Skill SDK

SDK для авторов MCP-скиллов платформы **Искра**: инструмент генерации `skillgen`,
рабочие примеры и референс. Скилл собирается в WebAssembly и исполняется в песочнице
хаба per-call.

## Быстрый старт
1. Установи инструмент: `go install github.com/iskradevs/skillsdk/cmd/skillgen@latest`
2. Создай каркас: `skillgen init my-skill` (или возьми за основу `examples/` — `units` минимальный, `weather` — с HTTP).
3. Опиши скилл в `manifest.yaml` (см. `reference/manifest.md`) и обнови обёртки: `skillgen gen --manifest manifest.yaml --out source`.
4. Реализуй `source/main.go` (`handle()` + хелперы), покрой `*_test.go`.
5. Проверь и упакуй: `skillgen validate manifest.yaml`, `skillgen check .` (опц.), `skillgen pack .`.
6. Загрузи `manifest.yaml` + `source.zip` на платформе.

Пошагово для coding-агента — в `AGENTS.md` / `CLAUDE.md`.

## Содержимое
- `cmd/skillgen` — генератор `mcphub_gen.go` из манифеста.
- `examples/` — рабочие скиллы (manifest + TinyGo-исходник + тесты).
- `reference/` — спека манифеста v1, 12 capabilities, frozen ABI, подмножество TinyGo.

## Лицензия
Apache-2.0 (см. `LICENSE`).
