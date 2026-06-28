# Справочник: manifest.yaml v1

Манифест описывает скилл-бандл — метаданные, источник кода, среду исполнения,
инструменты и политики доступа. Валидируется хабом при загрузке и перед сборкой.

Источник истины: `services/skillfactory/sdk/manifest/manifest.go` и `validate.go`.

---

## Верхний уровень: `Manifest`

```yaml
schema_version: 1          # обязательно; единственное допустимое значение — 1
bundle:     my-skill       # обязательно; короткое имя бандла
version:    1.0.0          # обязательно; semver (см. ниже)
author:     Иван Иванов
description: Что делает скилл
license:    MIT
source:     ...            # блок Source
runtime:    ...            # блок Runtime
tools:      []             # список Tool
acl:        ...            # блок ACL (см. примечание)
secrets:    []             # список Secret (опц.)
```

| Поле             | Тип     | Обязательно | Описание |
|------------------|---------|-------------|----------|
| `schema_version` | int     | да          | Версия схемы; хаб принимает 1..1 |
| `bundle`         | string  | да          | Имя бандла, не пустое |
| `version`        | string  | да          | Semver: `^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$` |
| `author`         | string  | нет         | Автор бандла |
| `description`    | string  | нет         | Описание назначения |
| `license`        | string  | нет         | SPDX-идентификатор лицензии |
| `source`         | Source  | да          | Откуда берётся исполняемый код |
| `runtime`        | Runtime | да          | Параметры исполнения |
| `tools`          | []Tool  | да          | Один или несколько инструментов |
| `acl`            | ACL     | см. прим.   | Политика доступа |
| `secrets`        | []Secret| нет         | Объявление secret-слотов |

---

## `Source`

```yaml
source:
  language:   go-tinygo          # или oci-image
  toolchain:  tinygo-0.41.1      # обязательно для go-tinygo
  # oci_image, oci_digest — только для oci-image
```

| Поле        | Обязательность | Описание |
|-------------|---------------|----------|
| `language`  | да | `go-tinygo` или `oci-image` |
| `toolchain` | для `go-tinygo` | Строка тулчейна; часть ключа кеша сборки. Допустимый charset: `^[A-Za-z0-9._-]+$` |
| `oci_image` | для `oci-image` | Ссылка на OCI-образ |
| `oci_digest`| для `oci-image` | Digest образа: `^(sha256:)?[0-9a-f]{64}$` |

Ограничение совместимости (кросс-валидация):
- `go-tinygo` → `runtime.kind` обязательно `wasm-per-call`
- `oci-image` → `runtime.kind` обязательно `container-long-running`

---

## `Runtime`

```yaml
runtime:
  kind:             wasm-per-call
  max_token_ttl_s:  300          # опц.
  quotas:
    timeout_s:      10
    memory_mb:      32
    cpu_ms:         1000
    interaction_timeout_s: 60    # обязательно при наличии request_input
```

| Поле                | Обязательность | Описание |
|---------------------|---------------|----------|
| `kind`              | да | `wasm-per-call` (go-tinygo) или `container-long-running` (oci-image) |
| `idle_timeout_s`    | для `container-long-running` | Время простоя контейнера; `>0` |
| `max_token_ttl_s`   | нет | Ограничение TTL токена; если задан — `>0`. `null`/отсутствие = хаб не ограничивает |
| `quotas`            | да | Блок Quotas |

### `Quotas`

| Поле                    | Обязательность | Правило |
|-------------------------|---------------|---------|
| `timeout_s`             | да | `>0`; wall-clock дедлайн всего вызова |
| `memory_mb`             | да | `>0`; для `wasm-per-call` не более 4096 (ограничение wazero) |
| `cpu_ms`                | да | `>0`; зарезервировано в Phase 1, но поле обязательно |
| `interaction_timeout_s` | да, если объявлен `request_input` | `>0`; максимальное время ожидания ответа пользователя |

---

## `Tool`

```yaml
tools:
  - id:          convert_units
    description: Переводит единицы измерения
    input_schema:
      type: object
      properties:
        value:  { type: number }
        from:   { type: string }
        to:     { type: string }
      required: [value, from, to]
    capabilities_required:
      - name: log
      - name: now
```

| Поле                   | Обязательность | Описание |
|------------------------|---------------|----------|
| `id`                   | да | Уникален в рамках бандла |
| `description`          | нет | Описание инструмента для LLM и UI |
| `input_schema`         | нет | JSON Schema (object) аргументов |
| `capabilities_required`| нет | Список Capability, необходимых этому инструменту |

Дублирующийся `id` в рамках одного манифеста → ошибка валидации.

---

## `Capability`

```yaml
capabilities_required:
  - name: http
    config:
      allow_hosts:
        - api.openweathermap.org
```

| Поле     | Описание |
|----------|----------|
| `name`   | Одно из 12 допустимых имён (см. раздел capabilities.md) |
| `config` | Конфигурация, специфичная для capability; для `http` — `allow_hosts` (обязательно, непустой список, wildcard `*` запрещён) |

Права проверяются per-tool при каждом вызове; наличие обёртки в коде не даёт доступа без объявления в манифесте.

---

## `ACL`

```yaml
acl:
  exposure_scope: instance   # instance | tenant | principal
  tenant_id:     ...         # обязательно при scope=tenant
  principal_id:  ...         # обязательно при scope=principal
```

| Поле             | Описание |
|------------------|----------|
| `exposure_scope` | `instance` — весь инстанс; `tenant` — одна организация; `principal` — один пользователь |
| `tenant_id`      | Обязателен при `exposure_scope: tenant` |
| `principal_id`   | Обязателен при `exposure_scope: principal` |

**Важное замечание для self-service загрузки.** Авторы скиллов **не пишут блок `acl:`** в манифесте.
Платформа проставляет `exposure_scope` автоматически при ingest:
- загрузка личного скилла → `principal`
- загрузка орг-скилла → `tenant`

Если автор явно задал `acl:` с конфликтующим `exposure_scope` — платформа вернёт 400.
Значение `instance` используется только в first-party embedded-pack бандлах.

> **В примерах SDK блока `acl:` нет** — они публикуются author-style. Значение
> `instance` существует только во внутренних first-party embedded-bundle и
> проставляется платформой при сборке, не автором.

---

## `Secret`

```yaml
secrets:
  - slot:        api_key
    description: Ключ API погоды
    scope:       openweathermap
    required:    true
```

| Поле          | Описание |
|---------------|----------|
| `slot`        | Имя слота; используется в вызове `Secret(slot)` |
| `description` | Подсказка пользователю при заполнении |
| `scope`       | Непрозрачное значение; семантику определяет backend секретов |
| `required`    | `true` — инструмент не активируется, если слот не заполнен |

---

## Примеры

### Минимальный манифест (без сети, только встроенные capabilities)

```yaml
schema_version: 1
bundle:       units
version:      1.0.0
author:       Искра
description:  Конвертация единиц измерения без внешних запросов
license:      MIT

source:
  language:  go-tinygo
  toolchain: tinygo-0.41.1

runtime:
  kind: wasm-per-call
  quotas:
    timeout_s: 5
    memory_mb: 16
    cpu_ms:    100

tools:
  - id: convert_units
    description: Переводит значение из одной единицы в другую
    input_schema:
      type: object
      properties:
        value: { type: number }
        from:  { type: string }
        to:    { type: string }
      required: [value, from, to]
    capabilities_required:
      - name: log
```

Рабочий образец: `examples/units`.

### HTTP-манифест (внешние запросы, секрет)

```yaml
schema_version: 1
bundle:       weather
version:      1.0.0
author:       Искра
description:  Текущая погода через OpenWeatherMap
license:      MIT

source:
  language:  go-tinygo
  toolchain: tinygo-0.41.1

runtime:
  kind: wasm-per-call
  quotas:
    timeout_s: 10
    memory_mb: 32
    cpu_ms:    500

secrets:
  - slot:        api_key
    description: OpenWeatherMap API key
    scope:       openweathermap
    required:    true

tools:
  - id: get_weather
    description: Возвращает текущую погоду для заданного города
    input_schema:
      type: object
      properties:
        city: { type: string }
      required: [city]
    capabilities_required:
      - name: http
        config:
          allow_hosts:
            - api.openweathermap.org
      - name: secret
      - name: log
```

Рабочий образец: `examples/weather`.
