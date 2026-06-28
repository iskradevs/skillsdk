# Поддерживаемое подмножество TinyGo (go-tinygo бандлы)

Источник: `services/mcphub/docs/tinygo-subset.md`

Производный документ (русский перевод с дополнениями для SDK). Не является
источником истины для parity-тестов — они сверяются с оригиналом.

---

## Тулчейн

**TinyGo 0.41.1** — первая ветка 0.41.x, собранная на Go 1.26.

Строка `tinygo-0.41.1` задаётся в поле `source.toolchain` манифеста и является
частью ключа кеша сборки: смена тулчейна инвалидирует все ранее собранные
артефакты.

---

## Флаги сборки (заморожены)

```
tinygo build -o out.wasm -target=wasm-unknown -buildmode=c-shared -scheduler=none -panic=trap .
```

| Флаг                       | Пояснение |
|----------------------------|-----------|
| `-target=wasm-unknown`     | Freestanding WebAssembly. **WASI выключен** — модуль не должен импортировать `wasi_snapshot_preview1`. Хаб не регистрирует WASI; любой WASI-импорт приводит к trap. Весь ввод-вывод, время и случайность проходят через `cap_call` (host capabilities), не syscall |
| `-buildmode=c-shared`      | *Reactor*-модуль: экспортирует `_initialize` вместо `_start`. Хаб вызывает `_initialize` однократно после инстанциирования, до первого вызова `handle` |
| `-scheduler=none`          | **Синхронный гость** — планировщик горутин отключён |
| `-panic=trap`              | Паника → wasm `unreachable` trap; WASI не задействован даже в panic-путях. Совпадает с `testdata/build-fixtures.sh` |

---

## Инварианты гостя

- **Только синхронность.** Нет горутин, каналов, `time.Sleep`. Запуск горутины из
  экспортируемой функции паникует (tinygo#3095). `handle` — чистый «запрос → ответ».
- **Нет WASI / нет syscall.** Время, случайность, логирование, HTTP, KV, секреты —
  только через `cap_call`. Прямой доступ к ОС недоступен по конструкции.
- **Per-call инстанс.** Свежий экземпляр модуля создаётся на каждый вызов и
  уничтожается после возврата `handle`. Ручной `free` в Phase 1 не требуется.
- **Квоты.** `timeout_s` (wall-clock) и `memory_mb` применяются хабом принудительно.
  `memory_mb` должен быть ≤ 4096 (ограничение wazero).

---

## Поддерживаемое подмножество stdlib (наблюдаемое)

Список растёт по мере валидации бандлов; всё, не указанное ниже, считается
неподдерживаемым до доказательства обратного.

### `encoding/json`

Работает (используется в echo-fixture). Ограничения рефлексии TinyGo:

- **Только именованные структуры.** Используйте именованные типы для всех
  JSON-объектов; анонимные inline-структуры не проходят рефлексию TinyGo.
- **`json.RawMessage`** — сквозной pass-through без промежуточной декодировки;
  работает корректно.
- **`map[string]string`** round-trip подтверждён сгенерированной fixture
  `genroundtrip.wasm` (codegen roundtrip test).
- Предпочитайте простые плоские структуры; избегайте `interface{}`, встроенных
  типов (embedded types) и глубоко вложенных дженериков.

### `unsafe`

- `unsafe.Slice` — использовать для чтения/записи буферов, переданных через `alloc`.
- `unsafe.Pointer` — использовать для преобразований указателей при работе с ABI.

### Прочее

- Базовые типы, слайсы, `map[string]string`, `map[string]json.RawMessage` — работают.
- Форматирование (`fmt.Sprintf` и базовые операции) — работает.
- Стандартные алгоритмы и математика — как правило работают; проверяйте на конкретных случаях.

---

## Генератор гостевого SDK

### `skillgen gen` (публичный SDK)

```bash
skillgen gen --manifest manifest.yaml --out source/
```

Генерирует `source/mcphub_gen.go`: frozen ABI boilerplate (`mcphub_abi_version`,
`alloc`, `cap_call` plumbing, `capInvoke`) плюс типизированные обёртки для объединения
`capabilities_required` по всем инструментам бандла.

Автор пишет только `func main() {}` (пустой) и `func handle(...)`.

Если `skillgen` не установлен:
```bash
go install github.com/iskradevs/skillsdk/cmd/skillgen@latest
```

### `mcphub gen` (платформенный инструмент, справочно)

```bash
mcphub gen --manifest manifest.yaml --out <dir>
```

Эквивалент для разработчиков платформы; генерирует тот же файл из того же набора
шаблонов (`sdk/codegen/templates/*.tmpl`). `skillgen` — публичный враппер над
той же логикой.

### Правила работы с `mcphub_gen.go`

- Файл помечен `// Code generated ... DO NOT EDIT`.
- Размещать рядом с `main.go` в одном `package main`.
- **Перегенерировать** после любого изменения `capabilities_required` в манифесте.
- Обёртка присутствует только для capabilities, объявленных хотя бы в одном
  `capabilities_required`; вызов необъявленной capability возвращает
  `capability_not_declared`.

---

## Дополнительные замечания

### Оператор: лимиты вызовов

Операторы могут ограничить число вызовов capability за один invoke через
`MCPHUB_CAPABILITY_CALL_LIMITS_JSON`. При превышении лимита — ошибка
`capability_call_limit_exceeded`; это бизнес-ошибка (завершите с тем, что есть),
не сбой инфраструктуры.

### Kill-gate (режим исполнения)

Хаб по умолчанию работает в режиме compiler (`wazero`). Если kill-gate
(бесконечный цикл при `timeout_s=1`) не срабатывает на целевой платформе,
установите `MCPHUB_WASM_MODE=interpreter` — в режиме интерпретатора
`WithCloseOnContextDone` форсирует проверку cancel.

### Производительность (Phase 1)

Phase 1 запускает **один свежий `wazero` runtime + `CompileModule` на каждый вызов**
(полная изоляция, максимальная простота). На darwin/arm64: ~58 мс в режиме compiler
против ~8 мс в интерпретаторе для echo+capcall; разница обусловлена JIT-компиляцией.
Кеш скомпилированных модулей по digest — плановая оптимизация (H3b+, `pool.go`).
