# UI-Engine — дизайн движка построения веб-интерфейсов

Ядро на **Go + WASM**, вся структура/темы/конфигурации/сеть/хуки/клавиши описываются **YAML**, всё поставляется как **модули**. Разработка — через **CLI**.

## Зафиксированные решения

| Область | Решение |
|---|---|
| Целевой рантайм | SPA в браузере, `GOOS=js GOARCH=wasm` |
| Модули | Файловые (папка/архив) |
| Формат модулей | Гибрид: wasm-логика + JS-шлюз для оберток (tiptap и т.п.) |
| Рендеринг | Virtual DOM с diff на Go |
| Реактивность | Полный VDOM diff (как React) |
| Ошибки/состояние JS-оберток | Go владеет состоянием, JS — чёрный ящик; синхронизация через JS-мост |
| CLI | Полный жизненный цикл + генерация кода из схем |
| Сеть | Только WebSocket, контракт в YAML (+ схемы + генерация кода) |
| Состояние | Декларативная модель состояния в YAML |
| Макет | Декларативное дерево в YAML (макеты + состояние) |
| Dev-цикл | Hot-reload (пересборка wasm + YAML) |
| Масштаб | Малая команда / соло |

---

## 1. Общая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                        БРАУЗЕР (SPA)                         │
│                                                              │
│   ┌──────────────────────────────────────────────────────┐   │
│   │              WASM-ядро (Go)                          │   │
│   │                                                    │   │
│   │  ┌──────────┐  ┌───────────┐  ┌──────────────────┐ │   │
│   │  │ Config   │  │ VDOM      │  │ Runtime          │ │   │
│   │  │ Loader   │→ │ + Diff    │→ │ (компоненты)     │ │   │
│   │  └────┬─────┘  └─────┬─────┘  └────────┬─────────┘ │   │
│   │       │             │                 │            │   │
│   │  ┌────▼─────┐  ┌─────▼─────┐  ┌───────▼───────┐    │   │
│   │  │ YAML     │  │ DOM       │  │ State         │    │   │
│   │  │ Parser   │  │ Renderer  │  │ (reactive)    │    │   │
│   │  └────┬─────┘  └─────┬─────┘  └───────┬───────┘    │   │
│   │       │             │                 │            │   │
│   │  ┌────▼─────┐  ┌─────▼─────┐  ┌───────▼───────┐    │   │
│   │  │ Module   │  │           │  │ WebSocket     │    │   │
│   │  │ Registry │  │ JS-Bridge │  │ Client        │    │   │
│   │  └────┬─────┘  └─────┬─────┘  └───────┬───────┘    │   │
│   └───────┼──────────────┼───────────────┼──────────────┘   │
│           │  fetch/wasm  │  JS calls     │  WSS            │
│   ┌───────▼──────────────▼───────────────▼──────────────┐  │
│   │                    JS Runtime (browser)             │  │
│   │         tiptap, другие внешние библиотеки, DOM      │  │
│   └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘

        │  сборка wasm + генерация кода
        ▼
┌───────────────────────────────────────────────┐
│                    CLI                       │
│  init · scaffold · build · dev(hot) · gen ·  │
│  lint · test · pub · version                 │
└───────────────────────────────────────────────┘
```

### Потоки данных

1. **Загрузка**: браузер получает HTML bootstrap → wasm-ядро монтируется → Module Registry загружает YAML-конфиги и модули → Config Loader строит дерево макета и состояния → VDOM рендерится в DOM.
2. **Интерактивность**: события из DOM/клавиатуры/сокета → событие в Runtime → мутация State → re-render затронутого поддерева через VDOM diff → патч DOM.
3. **Сеть**: WebSocket-клиент, управляется контрактом из YAML, обновляет State по схемам.
4. **JS-обертки**: JS-Bridge синхронизирует состояние между Go и внешними библиотеками (tiptap).

---

## 2. Роль YAML (единый источник правды)

YAML описывает **всё, что не является кодом модуля**:

| Файл / раздел | Назначение |
|---|---|
| `app.yaml` | Корневой конфиг приложения: монтирование модулей, точки входа, роуты |
| `screens/*.yaml` | Декларативное дерево макета: экраны → контейнеры → компоненты |
| `state.yaml` | Декларативная модель состояния: переменные, формы, списки |
| `theme.yaml` | Темы: токены (цвета, радиусы, отступы, типографика) + варианты (light/dark) |
| `net.yaml` | Контракт сокетного протокола: события, схемы, ошибки, реконнект |
| `hooks.yaml` | Веб-хуки: назначение внешних функций (HTTP/события) |
| `keys.yaml` | Назначения сочетаний клавиш → действия/события |
| `module.yaml` | Манифест модуля (см. §5) |

### Модель состояния (state.yaml)

```yaml
state:
  session:
    user: { type: object }          # binds из сокета
  form:
    login:
      email:   { type: string, required: true }
      password:{ type: string, secret: true }
  items:       { type: list }
  ui:
    drawerOpen: { type: bool, default: false }
```

### Макет (screens/login.yaml)

```yaml
screen: login
layout: column
children:
  - component: card
    props:
      title: "Вход"
    children:
      - component: input
        bind: state.form.login.email
        label: "Email"
      - component: input
        bind: state.form.login.password
        label: "Пароль"
      - component: button
        label: "Войти"
        on:
          click: [socket.send(auth.login), state.ui.loading=true]
  - component: alert
    bind: state.errors.login
```

Биндинг компонентов к путям состояния — это связка декларативного макета с реактивностью: при изменении `state.form.login.email` тронутое поддерево пере-рендерится.

---

## 3. Контракт сокетного протокола (net.yaml)

Только WebSocket. Описывается схемами, из которых CLI генерирует типизированный клиент и модели.

```yaml
net:
  url: "wss://api.example.com/ws"
  reconnect: { strategy: exponential, base: 500ms, max: 30s }
  auth:
    method: handshake
    tokenRef: state.session.token   # при получении handshake передается в auth
  events:
    client:                          # события, отправляемые клиентом
      auth.login:
        request:
          email:    string
          password: string
        reply: auth.result           # какое серверное событие ждем в ответ
      items.list:
        request:   { page: int, limit: int }
        reply:     items.list.result
    server:                          # события, принимаемые клиентом
      auth.result:
        payload:
          token: string
          user:  object
        on:
          - state.session.user = user
          - state.session.token = token
      items.list.result:
        payload: { items: [object], total: int }
        on: [state.items = items]
      items.updated:
        payload: { item: object }
        on: [state.items.update(item)]
      notify:
        payload: { message: string }
        on: [action.toast(message)]
errors:
  auth.required: { code: 401, on: [action.clearSession, screen.to(login)] }
  network:       { type: offline, on: [ui.toast("нет соединения")] }
ping:            { interval: 30s, timeout: 10s, onTimeout: reconnect }
```

- **Схема → код**: CLI генерирует Go-типы (`type AuthLoginRequest struct`), функции-отправки и обработчики с автобиндингом результата в State (`on:` маппит payload в пути состояния).
- **Валидация на рантайме**: входящий payload валидируется по схеме до того, как мутирует State.

---

## 4. Веб-хуки и сочетания клавиш

### hooks.yaml

```yaml
hooks:
  onReady:       [state.ui.ready=true, socket.connect()]
  onMountModule: [ ... ]
  onBeforeRoute: [ ... ]
  external:                      # вызовы внешних HTTP/JS-функций
    sendEmail: { endpoint: /mail, method: POST, schema: ... }
```

Хуки — декларативные «куда что вызывает»: на события жизненного цикла, на серверные события, на действия.

### keys.yaml

```yaml
keys:
  save:          { combo: [meta+s], action: [socket.send(items.save)] }
  openSearch:    { combo: [/, action: [screen.to(search), focus(input.search)] ] }
  toggleTheme:   { combo: [meta+shift+d], action: [theme.next()] }
  shortcutHint:  { combo: [shift+?], action: [action.shortcutsModal] }
```

Действия — это мини-DSL (см. §6), машина состояний клавиш централизована в Runtime.

---

## 5. Модульность

### Модуль = папка (или архив .uimod)

```
my-module/
├── module.yaml           # манифест
├── ui/
│   └── component.yaml    # декларативная структура компонента
├── go/                   # wasm-логика (опционально)
│   ├── component.go
│   └── go.mod
├── js/                   # JS-шлюз для оберток (опционально)
│   └── bridge.js
├── assets/
└── README.md
```

### module.yaml — манифест

```yaml
module:
  name: richtext
  version: 1.2.0
  type: component            # component | layout | wrapper | service
  entry:
    wasm: go/component.wasm   # необязательно
    js:   js/bridge.js        # необязательно (для оберток)
  declares:
    props:                    # контракт пропсов компонента
      value:  { type: string }
      onChange:{ type: event }
  depends:                    # зависимости модуля
    - editor-core ^5
  style: scoped               # scoped CSS или токены
```

### Обертка (wrapper) — tiptap-кейс

Wrapper = модуль с JS-шлюзом и декларативным интерфейсом, который оборачивает внешнюю JS-библиотеку. Состоянием владеет **ядро Go**, JS — чёрный ящик.

```yaml
# component.yaml (в модуле richtext)
component: richtext
wrapper: tiptap               # признак обертки над tiptap
host: js/bridge.js
props:
  value:    { type: string, bind: "allowed" }
  onChange: { type: event }
events:
  docChanged: { map: "value" -> state.document.body }
bridge:
  init: "TipTap.create(mount, opts)"      # инициализация в песочнице/parent
  get:  "editor.getHTML()"                # чтение из JS в Go
  set:  "editor.setContent(value)"        # запись из Go в JS
```

Жизненный цикл обертки:
1. Ядро монтирует контейнер в DOM.
2. JS-Bridge вызывает `init` — библиотека (tiptap) маунтится в контейнере.
3. JS-библиотека кидает события → bridge → ядро обновляет State (тихий путь, без опоры на DOM-события).
4. Если ядро меняет `value` из другого источника — bridge вызывает `set`, обновляя библиотеку.
5. Для избежания циклов: bridge маркирует «исходящее изменение» vs «входящее» (`source: go | js`).

---

## 6. Ядро (Go) — внутренние слои

Пакеты внутри wasm-бинаря:

```
core/
├── config/        # YAML-загрузка, валидация, схема конфига
├── module/        # Module Registry, разрешение зависимостей, версии
├── vdom/          # Virtual DOM: дерево узлов
├── diff/          # алгоритм сравнения VDOM (keyed, как React)
├── dom/           # рендерер патчей в браузерный DOM (syscall/js)
├── state/         # декларативная модель состояния + подписки
├── runtime/       # диспетчер событий, действия (мини-DSL), цикл
├── net/           # WebSocket client + контракт + валидация
├── bridge/        # JS-Bridge, callbacks + event loop
├── theme/         # токены, вариант, вычисление в CSS var
├── keys/          # машина сочетаний клавиш
├── hooks/         # выполнение веб-хуков
└── events/        # middleware, обработчики
```

### Реактивность (mini-DSL действий)

Действия из YAML (`on:` поля) — это маленький предметно-ориентированный язык, исполняемый runtime:

```
state.path = value        # запись в состояние (триггер re-render)
socket.send(evt, expr)    # отправка события по контракту
screen.to(name)           # навигация по экрану
action.toast(msg)
theme.next()
bridge.call(module, fn)   # вызов JS-обертки
if (cond) [ ... ]         # условия
```

Диспетчер событий — центральный: DOM-события, клавиши, хуки, сеть и JS-мост сходятся в один `event bus`, поэтому bиндинг единообразен.

### VDOM + Diff

- Дерево VDOM строится из YAML-макета + резолва компонентов модулей.
- Diff ключевой (с `key`), поддерживает списки (reorder без пересоздания).
- Преднамеренно простой, чтобы был предсказуемым в малой команде; без сложной структурной оптимизации React 18.

---

## 7. JS-Bridge (мост для оберток)

- **Сигнатура на границе**: Go-функции вызываются из JS и наоборот через `syscall/js`.
- **Callbacks + event loop**: JS-библиотека дергает заранее зарегистрированные функции-коллбеки; ядро ставит работу в свою очередь. Никаких блокирующих вызовов из JS в Go (async model).
- **Источник истины**: Go. JS-объект `window.UIEngine` — тонкий API (`mount`, `dispatch`, `render`), которым пользуются bridge.js и bootstrap.
- **Песочница для bridge.js**: модульный JS-шлюз исполняется в контексте страницы, но доступ к API ограничен `window.UIEngine` (+ явно разрешенные функции), чтобы не было полного доступа.

---

## 8. CLI (полный жизненный цикл)

```
ui-engine
├── init <dir>              # создать проект: скелет папок + app.yaml + default theme
├── module new <name>       # scaffold модуля (component/layout/wrapper/service)
├── module add <path|archive># подключить модуль в проект
├── build                   # скомпилировать wasm-логику, собрать bundle
├── gen                     # генерация кода из YAML-схем (модели, net-клиент, типы)
├── dev                     # dev-сервер: hot-reload (wasm + YAML), превью
├── lint                    # валидация YAML, схем, ссылок, целостности модулей
├── test                    # юнит-тесты логики модулей (go test wasm-forward)
├── export <archive>        # упаковать проект/модуль в .uimod
├── version                 # версионирование и changelog
└── serve                   # (опц.) статический сервер для собранного SPA
```

### Dev-цикл (hot-reload)

1. Watch на изменения в `**/*.yaml`, `go/**`, `js/**`, `assets/**`.
2. Изменение **YAML-слоя** (макет/тема/конфиг): перечитывается на лету в wasm, дифф рендерит изменения в живом DOM — **без пересборки**. (самый быстрый цикл)
3. Изменение **кода модуля** (Go/JS): пересборка wasm/bundle + мягкая переинициализация сессии (состояние презервуется где возможно).

---

## 9. Модули по умолчанию (стартовый набор)

| Модуль | Тип | Назначение |
|---|---|---|
| `layout` | layout | row/column/grid/stack, spacing, scroll |
| `text` | component | текст, heading, paragraph |
| `button` | component | кнопка + вариации (primary/secondary/ghost/danger) |
| `input/form` | component | поля, формы, валидация, автобиндинг |
| `list` | component | списки, пагинация, ключевой дифф |
| `card/alert/toast` | layout | составные контейнеры |
| `theme` | service | токены, light/dark, переключение |
| `error-boundary` | service | обработка ошибок рендера/сети |
| `richtext` (wrapper) | wrapper | обертка tiptap (пример) |

---

## 10. Дорожная карта (MVP → Production)

### Фаза 0 — Прототип (спайк)
- WASM-ядро: загрузка app.yaml, рендер статичного дерева (text/button) через VDOM.
- Go → DOM минимальный цикл.
- CLI: `init`, `build`, простой `dev` (полный reload).

### Фаза 1 — MVP
- Декларативная модель состояния + реактивность (подписки → re-render веток).
- Full VDOM diff, keyed-списки.
- WebSocket-контракт (net.yaml) + генерация клиента.
- hot-reload YAML (без пересборки) для макета/темы.
- CLI: `gen`, `lint`, `module new/add`.

### Фаза 2 — Обертки и мост
- JS-Bridge (callbacks + event loop).
- Модуль-обертка `richtext` поверх tiptap.
- Песочница для module JS.
- variation/темы light-dark.

### Фаза 3 — Production
- Webhooks, хуки жизненного цикла.
- Тесты, error-boundary, offline-поведение, реконнект-политики.
- `.uimod` экспорт/импорт, версионирование модулей.
- Возможность SSR-пререндера (как опция, заложить расширяемость). *(пока вне SPA-скоупа)*

---

## 11. Ключевые риски и открытые вопросы

1. **Генерация кода из YAML**: чему генерировать — только моделям сети или также типам state/maxeта? (Рекомендую: сети + типы состояния полностью; макет остаётся декларативным YAML без кода.)
2. **Граница «декларативно vs код»**: несмотря на «no-code», модули Go пишутся руками. Правило: *сборка приложения — YAML, переиспользуемая логика — модули (Go/wasm), внешние интеграции — обертки (JS)*.
3. **Производительность VDOM на больших списках** — решить стратегию (пагинация, виртуализация) в Фазе 2.
4. **Мини-DSL действий**: не перегрузить. Начать с подмножества (assignment, socket.send, screen.to, if/action), остальное — кодом модуля.
5. **Валидация YAML**: строгая схема конфигов (JSON Schema поверх YAML) в CLI `lint`, плюс рантайм-валидация в wasm.
6. **Мульти-вкладка/состояние сокета** — не цель для MVP.

---

## 12. Монорепо-структура проекта (предложение)

```
ui-engine/
├── DESIGN.md              # этот документ
├── core/                  # Go-ядро (был как пакеты, но под проект лучше отдельный модуль)
│   └── ...                # пакеты из §6
├── cli/                   # Go CLI (отдельный бинарь, native, не wasm)
├── runtime-js/            # bootstrap, window.UIEngine, bridge runtime, песочница
├── stdlib/                # модули по умолчанию (из §9)
│   ├── layout/
│   ├── button/
│   └── ...
├── tools/gen/             # кодогенератор (из YAML-схем)
├── examples/              # пример-приложение на движке
└── Makefile / Taskfile    # сборка
```
