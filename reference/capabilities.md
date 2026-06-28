# Справочник: capabilities (возможности)

Capability — именованная возможность, которую хаб предоставляет гостевому WASM-модулю
через host-функцию `cap_call`. Объявляется per-tool в `capabilities_required` манифеста;
наличие обёртки в коде **не** даёт права — хаб проверяет манифест при каждом вызове.

Источник: шаблоны `services/skillfactory/sdk/codegen/templates/*.tmpl` и
`services/mcphub/docs/tinygo-subset.md` (секция «Generated guest SDK»).

---

## Список capabilities (12 штук, v1)

| Имя             | Краткое описание |
|-----------------|-----------------|
| `http`          | Исходящие HTTP-запросы к разрешённым хостам |
| `now`           | Текущее время (unix nanoseconds) |
| `random`        | Криптографически стойкие случайные байты |
| `log`           | Структурированное логирование в хаб |
| `kv`            | Persistent key-value хранилище (per-bundle) |
| `secret`        | Получение значения именованного secret-слота |
| `llm`           | Запрос к языковой модели через LLM-шлюз платформы |
| `kb`            | Гибридный поиск по базе знаний вызывающего профиля |
| `files`         | Чтение KB-документов по ID |
| `progress`      | Однонаправленная трансляция прогресса выполнения |
| `show_view`     | Отображение декларативного UI (A4, однонаправленно) |
| `request_input` | Интерактивный запрос ввода у пользователя (A4, round-trip) |

---

## `http`

**Конфигурация манифеста:**
```yaml
capabilities_required:
  - name: http
    config:
      allow_hosts:
        - api.example.com
```

`allow_hosts` обязателен, непустой; wildcard `*` запрещён (anti-SSRF).

**Типы:**
```go
type HTTPRequest struct {
    Method  string            `json:"method"`
    URL     string            `json:"url"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    string            `json:"body,omitempty"`
}

type HTTPResponse struct {
    Status  int               `json:"status"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    string            `json:"body,omitempty"`
}
```

**Сигнатура обёртки:**
```go
func HTTP(in HTTPRequest) (HTTPResponse, error)
```

---

## `now`

**Конфигурация манифеста:** `config` не требуется.

**Сигнатура обёртки:**
```go
func Now() (int64, error)
```

Возвращает текущее время в unix nanoseconds.

---

## `random`

**Конфигурация манифеста:** `config` не требуется.

**Сигнатура обёртки:**
```go
func Random(n int) (string, error)
```

Возвращает `n` случайных байт, закодированных в hex-строку.

---

## `log`

**Конфигурация манифеста:** `config` не требуется.

**Сигнатура обёртки:**
```go
func Log(level, message string) error
```

`level` — произвольная строка (например `"info"`, `"warn"`, `"error"`).
Сообщение пишется в журнал хаба.

---

## `kv`

**Конфигурация манифеста:** `config` не требуется.

**Сигнатуры обёрток:**
```go
func KVGet(key string) (json.RawMessage, bool, error)
func KVPut(key string, value json.RawMessage) error
func KVDelete(key string) error
```

Persistent хранилище изолировано по бандлу. `KVGet` возвращает `(value, found, err)`;
`found=false` означает, что ключ не существует.

---

## `secret`

**Конфигурация манифеста:** `config` не требуется; слот должен быть объявлен
в блоке `secrets:` манифеста.

**Сигнатура обёртки:**
```go
func Secret(slot string) (value string, found bool, err error)
```

Возвращает значение secret-слота для вызывающего principal.
`found=false` — слот не заполнен (при `required: true` в манифесте инструмент
деактивируется ещё на этапе `tools/list`).

---

## `llm`

**Конфигурация манифеста:** `config` не требуется.

**Типы:**
```go
type LLMMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type LLMRequest struct {
    Messages []LLMMessage `json:"messages"`
}

type LLMResponse struct {
    Completion string `json:"completion"`
}
```

**Сигнатура обёртки:**
```go
func LLM(in LLMRequest) (LLMResponse, error)
```

Запрос идёт через LLM-шлюз платформы; модель и лимиты определяет оператор.

---

## `kb`

**Конфигурация манифеста:** `config` не требуется.

**Типы:**
```go
type KBChunk struct {
    ChunkID        string  `json:"chunk_id"`
    DocumentID     string  `json:"document_id"`
    DocumentName   string  `json:"document_name"`
    CollectionID   string  `json:"collection_id"`
    MimeType       string  `json:"mime_type"`
    Content        string  `json:"content"`
    Score          float64 `json:"score"`
    SectionContext string  `json:"section_context,omitempty"`
}

type KBResult struct {
    Results    []KBChunk `json:"results"`
    TotalFound int       `json:"total_found"`
}
```

**Сигнатура обёртки:**
```go
func KB(query string, limit int) (KBResult, error)
```

Гибридный поиск (semantic + keyword) по базе знаний, доступной вызывающему профилю
(profile-scoped). `limit=0` — сервер подставляет дефолт; ограничивается сервером сверху.

---

## `files`

**Конфигурация манифеста:** `config` не требуется.

**Типы:**
```go
type FileDocument struct {
    DocumentID   string `json:"document_id"`
    Name         string `json:"name"`
    MimeType     string `json:"mime_type"`
    CollectionID string `json:"collection_id"`
    Content      string `json:"content"`
    Truncated    bool   `json:"truncated"`
}

type FileDocumentMeta struct {
    DocumentID   string   `json:"document_id"`
    Name         string   `json:"name"`
    MimeType     string   `json:"mime_type"`
    CollectionID string   `json:"collection_id"`
    Tags         []string `json:"tags,omitempty"`
    ChunkCount   int      `json:"chunk_count"`
}
```

**Сигнатуры обёрток:**
```go
func FileRead(documentID string) (FileDocument, error)
func FileList(collectionID string) ([]FileDocumentMeta, error)
```

`FileRead` — полный текст KB-документа. `FileList` — список доступных документов;
`collectionID=""` возвращает все доступные. `FileDocument.Truncated=true` означает,
что содержимое было обрезано сервером.

---

## `progress`

**Конфигурация манифеста:** `config` не требуется.

**Сигнатура обёртки:**
```go
func Progress(progress, total float64, message string) (accepted bool, err error)
```

Однонаправленная трансляция прогресса (ADR-082). `total <= 0` — поле `total`
опускается. `message` обрезается сервером.

`accepted=false` — hint обратного давления: событие сброшено (throttling, нет
адресата). **Не ветвите** бизнес-логику на `accepted`: одно и то же
поведение даёт разные значения на разных инстансах (no-op sink всегда принимает).
Декларирование `progress` никогда не деактивирует инструмент — платформа без
progress-backend включает no-op sink.

---

## `show_view`

**Конфигурация манифеста:** `config` не требуется.

**Типы:**
```go
type ShowViewKV struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type ShowViewComponent struct {
    Type      string       `json:"type"`
    // text
    Value     string       `json:"value,omitempty"`
    // table
    Columns   []string     `json:"columns,omitempty"`
    Rows      [][]string   `json:"rows,omitempty"`
    // keyvalue
    Items     []ShowViewKV `json:"items,omitempty"`
    // list
    ListItems []string     `json:"list_items,omitempty"`
    Ordered   bool         `json:"ordered,omitempty"`
    // badge
    Text      string       `json:"text,omitempty"`
    Tone      string       `json:"tone,omitempty"`
}

type ShowViewSpec struct {
    Components []ShowViewComponent `json:"components"`
}
```

**Сигнатура обёртки:**
```go
func ShowView(spec ShowViewSpec) (accepted bool, err error)
```

Отображает декларативный UI пользователю (A4, однонаправленно). Доступные типы
компонентов: `text`, `table`, `keyvalue`, `list`, `badge`. Сервер строго проверяет
`Type`; неизвестный тип отклоняется. Поле списка — `list_items` (не `items`).

`accepted=false` — best-effort hint; не ветвите логику на нём.

---

## `request_input`

**Конфигурация манифеста:** `config` не требуется; **обязателен**
`runtime.quotas.interaction_timeout_s > 0` при объявлении этой capability.

**Типы:**
```go
type RequestInputKV struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type RequestInputComponent struct {
    Type      string             `json:"type"`
    Value     string             `json:"value,omitempty"`
    Columns   []string           `json:"columns,omitempty"`
    Rows      [][]string         `json:"rows,omitempty"`
    Items     []RequestInputKV   `json:"items,omitempty"`
    ListItems []string           `json:"list_items,omitempty"`
    Ordered   bool               `json:"ordered,omitempty"`
    Text      string             `json:"text,omitempty"`
    Tone      string             `json:"tone,omitempty"`
}

type RequestInputField struct {
    Name     string   `json:"name"`
    Type     string   `json:"type"`     // text | textarea | select | multiselect | confirm
    Label    string   `json:"label"`
    Required bool     `json:"required,omitempty"`
    Options  []string `json:"options,omitempty"` // для select/multiselect
}

type RequestInputSpec struct {
    Components []RequestInputComponent `json:"components,omitempty"`
    Inputs     []RequestInputField     `json:"inputs"`
}
```

**Сигнатура обёртки:**
```go
func RequestInput(spec RequestInputSpec) (values map[string]json.RawMessage, err error)
```

Приостанавливает скилл, отображает форму пользователю и блокирует выполнение до
получения ответа (round-trip, A4 Phase 2). Возвращает map `field_name → json.RawMessage`.
Ожидаемые типы значений по типу поля:
- `text`, `textarea`, `select` → `string`
- `multiselect` → `[]string`
- `confirm` → `bool`

Ошибка `input_timeout` возвращается, если пользователь не ответил за
`interaction_timeout_s` секунд.

---

## Общие замечания

- Обёртка в `mcphub_gen.go` генерируется только для capabilities, объявленных
  в `capabilities_required` хотя бы одного инструмента.
- Вызов undeclared capability возвращает ошибку `capability_not_declared`.
- Операторы могут ограничить число вызовов capability за один invoke через
  `MCPHUB_CAPABILITY_CALL_LIMITS_JSON`. При превышении лимита `cap_call` возвращает
  `capability_call_limit_exceeded` — это бизнес-ошибка, не сбой; завершите задачу
  с тем, что есть.
- `mcphub_gen.go` помечен `// Code generated ... DO NOT EDIT`; перегенерируйте
  командой `skillgen gen` после любого изменения `capabilities_required`.
