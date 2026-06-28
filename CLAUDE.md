# Сборка MCP-скилла для Искры — инструкция Claude Code

Ты — coding-агент (Codex/Claude Code). Твоя задача — собрать MCP-скилл из описания
пользователя и подготовить два файла для загрузки на платформу: `manifest.yaml` и
плоский `source.zip`. Платформа соберёт WASM на сервере — локально WASM собирать не нужно.

## Что такое скилл

Скилл — это набор инструментов (tools), скомпилированных в WebAssembly и исполняемых
в песочнице хаба per-call. Гость синхронный, без доступа к ОС: HTTP, время, секреты,
KV и т.п. идут через host-capabilities (обёртки в сгенерированном `mcphub_gen.go`).

## Структура каталога скилла

```
my-skill/
  manifest.yaml          # описание скилла (см. reference/manifest.md)
  source/
    main.go              # твой код: main() (пустой) + handle() + хелперы
    mcphub_gen.go        # СГЕНЕРИРОВАН skillgen — не редактировать
    go.mod               # module mcphubguest/<name>; go 1.26
    *_test.go            # опц. юнит-тесты чистых хелперов (под обычный go test)
```

## Шаги

### 1. Манифест
Заполни `manifest.yaml` по `reference/manifest.md`: `schema_version: 1`, `bundle`
(короткое имя), `version` (semver), `description`, `source` (`go-tinygo`,
`toolchain: tinygo-0.41.1`), `runtime` (`wasm-per-call` + quotas), `tools[]` с
`input_schema` (JSON Schema) и `capabilities_required`.

- Для каждого внешнего хоста добавь `capabilities_required: [{name: http, config:
  {allow_hosts: ["api.example.com"]}}]`. Wildcard `*` запрещён.
- Для секретов объяви `secrets:` (slot/description/scope/required) и используй
  capability `secret`.
- **Не пиши блок `acl:`** — scope (личный/орг) проставит платформа при загрузке.
  Явный конфликтующий `exposure_scope` будет отклонён (400).

### 2. Сгенерируй ABI-обёртки
Быстрее всего начать с `skillgen init <bundle>` — создаст каталог, заготовку
`manifest.yaml`, `go.mod` и сразу прогонит первый `gen`.
```
skillgen gen --manifest manifest.yaml --out source
```
Создаст `source/mcphub_gen.go` — frozen ABI (`alloc`, `handle`-плумбинг) + типизированные
обёртки под capabilities из манифеста (`HTTP`, `Now`, `Secret`, `KVGet`, …). Файл помечен
`DO NOT EDIT`; **перегенерируй после любой смены `capabilities_required`**.

Если `skillgen` не установлен:
`go install github.com/iskradevs/skillsdk/cmd/skillgen@latest`.

### 3. Напиши код
В `source/main.go` (`//go:build tinygo`, `package main`):
- `func main() {}` — пустой.
- `//go:wasmexport handle` `func handle(argsPtr, argsLen int32) int64` — разбери вход
  `{tool, args}`, сделай switch по `tool`, верни envelope через `writeJSON`.
- Результат — конверт `{"ok":true,"result":…}` или `{"ok":false,"error":{"code","message"}}`.
- Внешние вызовы — только через обёртки из `mcphub_gen.go`.

Смотри рабочий образец в `examples/weather/source/main.go`.

### 4. Тесты
Вынеси чистую логику (парсинг/форматирование) в отдельные функции и покрой обычными
`*_test.go` (без build-тега) — гоняй `go test ./source` обычным Go. См.
`examples/weather/source/format_test.go`.

### 5. (Опц.) Компиляционный smoke под TinyGo
Если установлен TinyGo 0.41.1:
```
skillgen check .
```
`check` делает это за тебя; сырые замороженные флаги — в `reference/abi.md`.

### 6. Проверь и упакуй
```
skillgen validate manifest.yaml   # манифест валиден, без acl
skillgen check .                  # опц.: локальная сборка под TinyGo (если установлен)
skillgen pack .                   # → source.zip (плоский, без *_test.go, с go.mod)
```
`pack` сам собирает плоский zip и переименовывает `go.mod.txt`→`go.mod` — руками
`zip` звать не нужно.

### 7. Загрузка
Отдай пользователю `manifest.yaml` + `source.zip`. Он грузит их на платформе
(вкладка «Загрузить готовое»). После модерации скилл появится выключенным —
пользователь включит тумблером.

## Чек-лист и частые ошибки
- WASI выключен (`-target=wasm-unknown`): никаких syscall/файлов/сети напрямую — только обёртки.
- Гость синхронный: **нет** goroutine, channel, `time.Sleep` (паника/trap).
- Только именованные структуры (ограничения рефлексии TinyGo); избегай анонимных inline-структур.
- `capabilities_required` — per-tool: обёртка в коде не даёт права, хаб проверяет манифест.
- `capability_call_limit_exceeded` — это бизнес-ошибка (заверши тем, что есть), не сбой.
- Полная спека и список 12 capabilities — в `reference/`.
