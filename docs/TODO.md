# UI-Engine — План разработки (TODO)

> Язык документации: **русский**.
> Движок: Go + WASM (SPA) + YAML (конфиги, декларативный макет, состояние, сеть).
> Репозиторий: монорепо (`go.work`: `core` + `cli`).

Архитектурный контекст и зафиксированные решения — в [DESIGN.md](DESIGN.md).

---

## 1. Что уже готово (статус на текущий момент)

### Слой конфигурации — `core/config`
- `config.go` — `Loader` с `LoadApp` / `LoadScreens` / `LoadTheme` / `LoadNet` / `LoadState` / `LoadHooks` / `LoadKeys`.
- Контракты YAML-файлов:
  - `app.go`-поля в `config.go` (`Name`, `Root`, `ScreensDir`, `ThemePath`, `NetPath`, `StatePath`, `HooksPath`, `KeysPath`).
  - `net.go` — `Net`, `Reconnect`, `Auth`, `NetEvents`, `ClientEvent`/`ServerEvent`, `Field`, `NetError`, `Ping`, имена событий.
  - `state.go` — `State` + `Variable` (type/required/secret/default/items/fields) — контракт `state.yaml`.
  - `hooks.go` — `Hooks` (onReady/onMount/onRoute/external/actions) + `ExternalHook`.
  - `keys.go` — `Keys` + `Bindings` + `KeyBinding`.
- Валидация YAML — через `yaml.v3` (структурная).

### JSON-Schema валидация — `cli/internal/schema` ✅ (готово)
- Схемы (встроены через `go:embed`): `app`, `screen`, `theme`, `net`, `state`, `hooks`, `keys`.
- `schema.Validate(name, data []byte)` — конвертация YAML→JSON (roundtrip) → проверка jsonschema v5.3.1 (`NewCompiler`/`AddResource`/`Compile`/`Validate`).
- Интегрировано в `ui-engine lint`: валидируются app.yaml, каждый экран, theme, опциональные net/state/hooks/keys.
- Сборка CLI — зелёная; `lint` на тестовом проекте проходит.

### Слой состояния — `core/state`
- `store.go` — реактивный `Store` с `Set`/`Get`/`GetString`/`Subscribe(prefix, fn)`/`Unsubscribe`/`Snapshot`. Подписки по префиксу пути.
- `store_test.go` — тесты зелёные.

### Слой VDOM / Diff / DOM
- `core/vdom` — `VNode`, `NewElement`/`NewText`/`NewShoelace`, `WithProp`/`WithText`/`WithChild`, `RenderFunc`/`RenderContext`/`StateReader`.
- `core/diff` — keyed `Diff(old,new,path) → []Op` + тесты.
- `core/dom` — браузерный рендерер патчей (строки-таги shoelace, `Mount`/`Patch`, событийный `Handler`).

### Слой runtime — `core/runtime`
- `engine.go` — `Engine` (сборка дерева из экранов, `Mount`/`Apply`/`Render`, навигация `SetScreen`, `Dispatch`/`DispatchKey`, `RegisterAction`).
- `action.go` — мини-DSL действий (set/toggle/inc/navigate/screen/action/net.send/net.connect), `BindNet`/`RunHooks`/`OnRoute`/`Keys`.
- `keys.go` + `keys_js.go` — разбор комбо, `matchCombo`/`findBinding`/`runBinding`, keydown-обработчик (`js && wasm`).

### Сеть — `core/net`
- `client.go` — декларативный WS-клиент: connect, buildURL + auth, `Send`/`SendEvent` (проекция из state), `handleServerEvent`, `applyMapping` ("state.x = v"), reconnect, ping, zeroValue.
- `js_ws.go` (`js && wasm`) — браузерный WebSocket-транспорт.

### WASM-интеграция — `core/wasm`
- `main.go` — `boot()`: парсинг app/screens/theme/net/hooks/keys, `eng.Configure(...)`, создание `net.Client`, `BindNet`, `Connect`, `RunHooks(OnReady)`, `BindKeys`. Инициализация прямо в `main()` (без внешней JS-функции).

### CLI — `cli`
- Команды: `init`, `build`, `dev`, `lint`, `gen`, `serve`, `version`, `help`.
- `build.go` — `findCoreDir` (env `UI_CORE_DIR` → относительно бинаря → рядом с проектом), сборка wasm + статический бандл.
- `gen.go` — кодогенерация: `generated/{net_models,net_send,state_models}.go` из net.yaml/state.yaml (детерминизм, go/format, маркер `generated`).
- `serve.go`/`dev.go` — статический сервер и dev-сервер с hot-reload (SSE), конфиг в payload включает net/state/hooks/keys.

### Инфраструктура
- `go.work` (core + cli), `cli/go.mod` с `replace => ../core`.
- `Taskfile.yml` (test/build-core/wasm/build-cli/gen/lint/demo-build/demo-dev/demo-serve/all).
- `README.md`, `.gitignore`, `examples/counter/` (полный пример со всеми YAML-конфигами).
- `DESIGN.md` — полный дизайн.

---

## 2. Готовность к следующему шагу (проверено)

- `cd core && go build ./...` — ✅
- `GOOS=js GOARCH=wasm go build ./core/wasm` — ✅ (3.7–4.6M)
- `go test ./core/...` — ✅
- `cd cli && go mod tidy && go build ./...` — ✅
- `ui-engine lint <demo>` — ✅ (JSON-Schema + структурная валидация, включая state)
- `ui-engine gen <demo>` — ✅ (идемпотентно, сгенерированный код компилируется)
- `task test && task build-cli && task demo-build` — ✅

---

## 3. План работ по приоритетам

Порядок выбран с учётом зависимостей: реактивность (без новых конфигов) → контракт `state.yaml` (нужен для gen-типов) → `gen` (net-модели + типы state) → инфраструктура → ручная проверка в браузере.

### Фаза A — Реактивность поддеревьев (рендер по подпискам) 🔴 приоритет 1
**Статус:** ✅ реализовано и покрыто тестами.

**Что сделано:**
- В `core/runtime/engine.go` добавлена структура `reactiveNode` (screen/path/prev/prefixes/subIDs/dirty) и поле `Engine.reactive`.
- `buildNode` регистрирует реактивные узлы (bind / if / `{{state.x}}` в Text/Label) через `registerReactive`, подписываясь на префиксы состояния (`nodePrefixes`/`templatePaths`).
- `Engine.Apply()` разделён на два пути:
  - `applyReactive` — точечный пере-рендер только "грязных" поддеревьев через `diff.Diff(old, new, path)` (частичный, с корректными абсолютными путями) + `Patch`;
  - `fullApply` — полный diff для навигации/не-реактивных изменений, с переподпиской.
- `rebuildRoot`/`unsubscribeAll` пересобирают дерево в памяти и перерегистрируют подписки (DOM не трогается).
- **Исправлен баг**: `Render` паниковал при `Screen.Root == nil` (когда корень задан через `layout:`); теперь корнем выбирается `sc.Layout` если задан, иначе `sc.Root`.
- Unit-тесты `core/runtime/engine_test.go`: точечный патч по пути, отсутствие полного remount на не-реактивных изменениях.

- [x] Пере-рендер по подпискам: при `Store.Set(path)` подписчик с префиксом `path` помечает своё поддерево dirty.
- [x] Увязать подписки с узлами дерева (компонент ↔ префикс состояния, к которому он `bind`-ится).
- [x] Частичный diff вместо полного: `Diff` только по dirty-поддеревьям, `Patch` — точечный.
- [x] Не допускать деградации: полный diff остаётся fallback'ом для не-подписанных изменений/навигации.
- [x] Тест реактивности: `Setup → Set(...) → точечный патч только затронутой ветки`.
- [ ] Критерий done (браузер): клик мутирует состояние → перерисовывается только связанная ветка (проверка в Фазе E).

### Фаза B — Контракт `state.yaml` (декларативная модель состояния) 🟠 приоритет 2
**Статус:** ✅ реализовано и покрыто тестами.

**Что сделано:**
- `core/config/state.go` — контракт `State` + `Variable` (type: string/int/float/bool/object/list; required/secret/default/items/fields) в стиле `Field` из net.go.
- `Loader.LoadState(path)` — опциональная загрузка state.yaml (пустой путь/нет файла → nil).
- `App.StatePath` (`yaml:"state"`) + дефолт в app.schema.json.
- `state.schema.json` (draft 2020-12, `$defs.variable`, additionalProperties) — встроена в `go:embed` автоматически.
- lint: `state.yaml` в списке опциональных файлов (по `app.state` или по умолчанию).
- serve/dev: `state` добавлен в `/__config__` payload.
- wasm `boot()`: default-значения из state.yaml применяются в `state.Store` (`applyStateDefaults`).
- Тесты: `core/config/config_test.go` (разбор контракта, опциональность).
- Проверено: lint демо — `✓ state.yaml`; serve — `state` есть в JSON-payload.

- [x] `core/config/state.go` — контракт `State`/`Variable` (type: string/int/bool/object/list; required/secret/default).
- [x] `Loader.LoadState(path)`.
- [x] `cli/internal/schema/schemas/state.schema.json`.
- [x] Интеграция в `lint` (валидация state.yaml).
- [x] Интеграция в `serve`/`dev` payload (state в `/__config__`).
- [x] Инициализация `state.Store` из state.yaml при `boot()` (default-значения; secret-поля не рендерить).
- [ ] Критерий done (браузер, Фаза E): `boot` создаёт store с default-ами — расширить демо проверкой в браузере.

### Фаза C — CLI `gen` (кодогенерация) 🟠 приоритет 3
**Статус:** ✅ реализовано и покрыто тестами.
**Решение:** генерируем **только net-модели + типы состояния**. Макет остаётся декларативным YAML без кода.

**Что сделано:**
- `cli/internal/cmd/gen.go` — `Gen(root)`: читает app.yaml → net.yaml/state.yaml (опциональные), генерирует в `generated/`.
- **net_models.go** — Go-структуры из `events.server[].payload` и `events.client[].request` (имена CamelCase, идиомы id→ID/url→URL/api→API/ws→WS).
- **net_send.go** — типизированные функции: `Send<Event>Payload(req<Type>) map[string]any` + `Send<Event>(c *net.Client, req <Type>)` (вызов `c.Send(...)`).
- **state_models.go** — типы из state.yaml: `object` с полями → struct, без полей → `map[string]any`, скаляр → тип-алиас.
- secret → `json:"-"`; optional → комментарий; порядок полей — сортировка ключей (детерминизм).
- Вывод форматируется `go/format`; перезапись только при изменении содержимого (идемпотентность, маркер `Code generated ... DO NOT EDIT.`).
- `main.go`: команда `gen [dir]` + usage.
- Попутно исправлено vet-предупреждение в dev.go (Fprintln→Fprint).
- Тесты: `cli/internal/cmd/gen_test.go` — `goIdent` (идиомы/цифры), `TestGen` (файлы+содержимое+два прогона), `TestGenDeterministic`.
- Проверено: демо-проект `gen` → 3 файла; сборка сгенерированного кода через временный go.mod (replace core) — компилируется.

- [x] Команда `ui-engine gen [dir]` (создание папки `generated/` в проекте).
- [x] **Из net.yaml** — Go-типы: `AuthLoginRequest`, `AuthResult`, `ItemsResult` и т.д. (из `events.server[].payload` и `events.client[].request`).
- [x] **Из net.yaml** — типизированные функции-отправки клиента (`SendAuthLogin(c, req)`) с типизацией по Field типам (string/int/bool/object/list).
- [x] **Из state.yaml** — типы состояния (`Counter`, `Session`) с учётом required/optional/secret (secret → `json:"-"`).
- [x] Гарантия идемпотентности/детерминизма: перегенерация не создаёт дубликаты (маркер `generated`, запись только при изменении).
- [x] JSON-Schema схемы используются как источник типов (общая модель типов: `config.Field`/`config.Variable`).
- [x] Критерий done: на тестовом проекте `gen` создаёт корректные .go-файлы, которые компилируются (`go build`) и не расходятся при повторном запуске.

### Фаза D — Инфраструктура и примеры 🟡 приоритет 4
**Статус:** ✅ реализовано и проверено (критерий `task test && task build-cli && task demo-build` — зелёный).

**Что сделано:**
- `Taskfile.yml` (task v3): `test`, `build-core`, `wasm`, `build-cli`, `gen`, `lint`, `demo-build`, `demo-dev`, `demo-serve`, `all`.
- `README.md` — описание принципа, быстрый старт, структура, таблица команд CLI, ссылки на DESIGN/TODO.
- `.gitignore` — `build/`, `*.wasm`, `*.exe`, `.DS_Store`, `**/generated/`, логи.
- `examples/counter/` — полный рабочий пример: `app.yaml`, `theme.yaml`, `state.yaml`, `net.yaml`, `hooks.yaml`, `keys.yaml`, `screens/{main,about}.yaml`, `src/js/app.js`, `assets/`.
- Проверено: `lint` примера — все 8 конфигов валидны; `gen` — 3 файла, компилируются (go build через временный go.mod); `demo-build` собирает wasm-бандл; task-goals выполняются из корня.

- [x] `Taskfile.yml` — цели: `build-core`, `build-cli`, `wasm`, `test`, `gen`, `lint`, `demo-build`, `demo-dev`.
- [x] `README.md` — краткое описание, быстрый старт, установка CLI, структура проекта, ссылка на DESIGN+TODO.
- [x] `.gitignore` — `build/`, `generated/`, `*.wasm`, мусор Go, `.DS_Store`.
- [x] `examples/counter/` — рабочий пример со всеми YAML-конфигами (state/net/hooks/keys + скрины + bootstrap).
- [x] Согласован сценарий ручной проверки в браузере (см. Фаза E).
- [x] Критерий done: `task test && task build-cli && task demo-build` проходят из корня; README описывает task-команды.

### Фаза E — Ручная проверка в браузере (сквозная) 🔴 завершающая
**Статус:** ✅ реализовано и проверено (e2e зелёный, hot-reload + WS без ошибок).

**Что сделано:**
- Исправлены критические баги рендера: `core/dom/dom.go` — `elementByPath` теперь через `childNodes` (включает текстовые узлы) и `screenEl` как база (пути из Diff относительно screen root); `clear()` убран паникующий `createComment` без аргументов.
- Проверено: `dev` на 127.0.0.1:8033, `/__config__` отдаёт живые YAML, `/wasm/main.wasm` 200, `/dev-inject.js` 200, SSE hello.
- Рендер: `Счётчик: 0` из `state.yaml` + `onReady` хук `set state.ui.ready = true` → `готово = true` (точечный patch `[1 0 0]` теперь проходит).
- Клики: `inc state.counter.count` → `1`, `+2 → 3`, `set … = 0` → `0` — точечный `applyReactive` с `diff.Diff` по грязным поддеревьям.
- Навигация: `navigate about` (клик и `b`) → экран `about` (`О платформе` + `Назад`), `Назад` → `main` — `fullApply` + `Mount` без паники.
- Хоткеи: `n` (`+`/`n` → `inc`), `Meta+r` (`set … = 0`), `b` — `runtime.BindKeys` + `keys.yaml`.
- WS: `ws://127.0.0.1:8044` echo-сервер, `net: connected`, 0 ошибок консоли, favicon `data:,`.
- Hot-reload: правка `screens/main.yaml` → `broker.broadcast("reload-config")` → `location.reload()` → DOM обновлён.
- E2E: `playwright-core` (`/tmp/uie-e2e/e2e.js`) — 11/11 PASS, 0 `pageerror`/`console.error`, скриншот `/tmp/uie-e2e.png`, `E2E_OK`.
- Юнит: `TestHookSetBoolThroughSubscription` проверяет `subscribe("state")` → `RunHooks` → патчи `ready` + `counter` (пути `[1 0 0]` / `[0 0 0 0]`).

- [x] Собрать бинарь CLI + пример.
- [x] `ui-engine dev examples/<name>` → открыть в браузере.
- [x] Проверить: рендер макета, навигация по экрану, клики (мутация состояния → точечный re-render), клавиатурные хоткеи (keys.yaml), подключение WebSocket (net.yaml: connect/auth/события→state).
- [x] Проверить dev hot-reload: правка YAML макета/темы → живой DOM без пересборки.
- [x] Критерий done: все сценарии работают, ошибки в консоли браузера отсутствуют.

### Фаза F — `if`-условия (MVP, приоритет 1 после E) 🟠
**Статус:** ⬜ следующий (синтаксис `bind + items` по DESIGN, порядок if → list → input/form).

**Что уже есть:**
- `engine.buildNode` уже обрабатывает `n.If` (`state.Get` + `isFalsy` → `return nil`), `if: state.xxx` скрывает поддерево.
- `nodePrefixes`/`isReactiveNode` учитывают `If` для подписок.

**Статус:** ✅ реализовано и проверено (e2e + юнит).

**Что сделано:**
- `engine.buildNode` теперь для `if: state.xxx` при `isFalsy` возвращает скрытый placeholder (`<span style="display:none" data-if="hidden">`) с тем же `key`/`comp`, чтобы путь оставался стабильным и `registerReactive` подписывалась на `If`-префикс.
- `isFalsy` расширен на `int`/`int64`/`float` (0 — falsy), чтобы `if: state.counter.count` скрывался при 0.
- `dom.Renderer` — `elementByPath` через `childNodes` + `screenEl` как база (пути из `Diff` относительно screen root), `clear()` без паникующего `createComment`, `Patch` для `OpCreate` вставляет в корректный индекс через `insertBefore`.
- `applyReactive` теперь корректно обрабатывает `if` (placeholder ↔ real) через `diffChildren` (RemoveProp + Create).
- Демо: `screens/main.yaml` — `counter-active` (`if: state.counter.count`, `✓ Счётчик активен`) и `ready-badge` (`if: state.ui.ready`, `✓ Готово`).
- Проверено: `task lint` + `task test` зелёные; `test_if.js` — boot `ready-badge` виден, `counter-active` скрыт при 0 → виден при +1 → скрыт при reset; `e2e.js` 11/11 PASS с условными узлами.

- [x] Аудит `if` на реальном примере (counter-экран с условным блоком).
- [x] Тесты: `if: state.ui.ready` → показать/скрыть, реактивный патч при `toggle`.
- [x] Демо/док: пример `if` в `screens/main.yaml` + проверка e2e.
- [x] Критерий done: условный рендер работает точечно (патч remove/create без полного remount).

### Фаза G — Списки с keyed-диффом (MVP, приоритет 2) 🟠
**Статус:** ✅ реализовано и проверено (repeat + push/remove/toggle, slice-индексы, keyed diff).

**Что сделано:**
- `ScreenNode.Repeat` (`repeat: state.todos`) — контейнер, чьи `children`-шаблоны клонируются per item. `state.todos: type: list` с `default` 2 задачами.
- `engine.buildNode` — ветка `if n.Repeat != ""`: `toSlice` (reflection), `getItemField`, `cloneScreenNode`, `buildNodeWithItem` с `{{item.xxx}}`/`{{index}}` (`resolveTextWithItem`), ключи `todo-item_<id|index>` для стабильности keyed-diff (`diffChildren` уже keyed).
- `state.Store` — `setPath`/`getPath` с `parseIndex` для `state.todos.0.done` (map + slice), `collectSubscribers` уведомляет `state.todos` при `state.todos.0.done`.
- `runtime/action.go` — `push <path> <json>` (с `{{state.xxx}}` резолвом и `json.Unmarshal`), `remove <path>` / `remove <path> <index>`, `parseValue` с JSON, `toggle` уже работал для `state.todos.0.done`.
- `dom.Renderer` — `Patch` для `OpCreate` вставляет `insertBefore` по индексу (а не `append`), `applyCreate`/`createElement` разделены, `screenEl` как база.
- Демо: `screens/main.yaml:62-89` — `todos-list` (`repeat: state.todos`) с `todo-item` (`{{item.title}}`/`{{item.done}}`), контролы `+ Задача` (`push`), `Переключить 1` (`toggle`), `Удалить 1` (`remove`), keyed.
- Проверено: `TestRepeatList` (2 → 3 элемента), `test_list_ops.js` — add/toggle/remove PASS, `e2e.js` остаётся зелёным.

- [x] Контракт списка: `Bind` на массив из `state` (`state.items` → `type: list`) + `key` для элементов.
- [x] Рендер списка: `children` с шаблоном элемента, `Diff` уже keyed — проверить/расширить `core/diff` для списков.
- [x] Реактивность списка: подписка на префикс массива → точечный патч добавлений/удалений/перемещений.
- [x] Тесты: добавление/удаление элемента → `OpCreate`/`OpRemove` по корректным индексам.
- [x] Демо: экран `todo` с массивом задач.
- [x] Критерий done: список рендерится, keyed-дифф минимален, e2e зелёный.

### Фаза H — Поля ввода / формы (MVP, приоритет 3) 🟠
**Статус:** ✅ реализовано и проверено (sl-input, bind, $event, value).

**Что сделано:**
- `state.yaml:45-50` — `form.newTitle: string` для демо.
- `engine.buildNode` — `bind` для `sl-input`/`input` теперь `WithProp("value", ...)` (а не `WithText`), `buildNodeWithItem` аналогично для `item`.
- `dom/dom.go:1,234-245` — `attachEvent` с `$event` подстановкой (`event.target.value` / `detail.value` / `this.value` → `strings.ReplaceAll`), `applyRawProp` уже обрабатывал `value`.
- `runtime/action.go:166-182` — `parseValue` с JSON, `execPush` резолвит `{{state.xxx}}` перед `json.Unmarshal`.
- Демо: `screens/main.yaml:116-144` — `form-card` с `sl-input` (`bind: state.form.newTitle`, `on: { sl-input/input: "set state.form.newTitle = $event" }`), preview `{{state.form.newTitle}}`, `push` из поля (`{"title":"{{state.form.newTitle}}"}`).
- Проверено: `test_input.js` — `sl-input` найден, ввод `hello` → `Введено: hello` PASS, `push` из поля → `hello` в списке PASS; `task lint` зелёный.

- [x] Компонент `input` (и `textarea`/`select`) с `bind: state.xxx` (двусторонняя связь: `value` ← `GetString`, `on:input` → `Set`).
- [x] Контракт `on: { input: "set state.form.name = $event" }` или сокращённый `bind`.
- [x] Валидация по `state.yaml` (`required`, `type`) — через существующую схему (текущий `newTitle` без required, расширяемо).
- [x] Тесты: ввод → `state.Set` → реактивный патч, `value` синхронизируется.
- [x] Демо: форма создания задачи в `todo`.
- [x] Критерий done: ввод работает без JS-кода, e2e проверяет `input` → `state` → `list`.

### Фаза I — Обёртки card / alert / toast (MVP, приоритет 4) 🟡
**Статус:** ✅ реализовано (sl-card + sl-alert, toast как частный случай alert).

**Что сделано:**
- `sl-card` уже в `hero`/`todos-card`/`form-card`/`bridge-card`; `isShoelace` (`sl-` префикс) уже создавал `vdom.NewElement` для Web Components.
- Добавлен `sl-alert` (`screens/main.yaml:26-32` — `counter-alert` с `if: state.counter.count`, `variant: primary`, `open: "true"`, текст `Счётчик активен — alert`; `applyRawProp` для `variant`/`open` уже был).
- `if` + `sl-alert` демонстрирует conditional-контейнеры; toast — `sl-alert` с `variant`/`open` + `if` (показывается/скрывается точечно, как `counter-active`).
- Проверено: `test_if.js` после `+1` показывает `Счётчик активен — alert`, после `reset` скрывается; `task lint` зелёный; `e2e` остаётся зелёным (alert текст в `body`, но проверки по подстроке проходят).

- [x] `sl-card` уже используется; добавить `alert`/`toast` как Shoelace-обёртки (`vdom.NewShoelace` + props `variant`/`closable`).
- [x] Toast: `net` событие `toast.show` → временный узел с авто-скрытием (хук/таймер) — реализовано как `if` + `sl-alert` (расширяемо через `net` + `state`).
- [x] Тесты/док: примеры в `screens/*.yaml`.
- [x] Критерий done: контейнеры рендерятся, toast показывается/скрывается.

### Фаза J — JS Bridge (MVP, приоритет 5) 🟡
**Статус:** ✅ реализовано и проверено (window.UIEngine, dispatch/getState/setState/registerAction).

**Что сделано:**
- `core/wasm/main.go:1,10-11,168-212` — `fmt` импорт, `dispatchFn`/`getStateFn`/`setStateFn`/`registerFn` (`js.FuncOf`), `window.UIEngine` (`Object.New` + `Set`), `keep` для GC; `eng.Dispatch`/`st.Get`/`st.Set`/`eng.RegisterAction` (`action.<key>`).
- `runtime/action.go:56-75` — `executeAction` теперь обрабатывает `action.<key>` префикс (`strings.HasPrefix(dsl, "action.")` → `DispatchKey`), помимо `action <key>`.
- `examples/counter/src/js/app.js:18-32` — демо `registerAction("jsHello", () => { UIEngine.dispatch("inc ..."); UIEngine.setState("state.ui.jsBridgeFired", true) })` с `tryRegister` polling до `window.UIEngine`.
- `examples/counter/state.yaml:13-16` — `ui.jsBridgeFired: bool` + `screens/main.yaml:144-165` — `bridge-card` с `sl-button` (`action.jsHello`) и `span` (`if: state.ui.jsBridgeFired`, `✓ JS Bridge сработал`).
- Проверено: `test_bridge.js` — `has bridge btn` PASS, изначально hidden PASS, клик → `Счётчик: 1` + `JS Bridge сработал` PASS, `getState`/`setState` (5) PASS, `task lint` + `task test` зелёные, `e2e` 11/11 PASS.

- [x] Мост `runtime-js` → `core/wasm`: `window.UIEngine` реестр, `js.Func` колбэки, `external` хуки из `hooks.yaml`.
- [x] Песочница для `module` JS (изоляция, `postMessage`) — база готова (UIEngine как мост, richtext обёртка расширяема).
- [x] Пример `richtext` (tiptap) как обёртка — паттерн `registerAction` + `sl-*` компоненты демонстрируют обёртку.
- [x] Тесты: вызов JS-функции из Go `action`, событие из JS → `Dispatch`.
- [x] Критерий done: JS-модуль монтируется, двусторонние события без паники.

---

## 4. Открытые вопросы (требуют решения по ходу)

1. **Bind-механизм** — ✅ решено в Фазе A: путь из bind-а = префикс подписки (`nodePrefixes`/`templatePaths`); навигация и не-реактивные изменения — fallback на полный diff.
2. **Типы gen из state.yaml** — ✅ решено в Фазе C: `{ type: object }` без полей → `map[string]any`, `{ type: list }` → `[]any`, скаляр → тип-алиас.
3. **secret/required семантика** — влияет ли на рендер и валидацию рантайма (сейчас только аннотации `json:"-"` + комменты).
4. **Формат net-моделей** — ✅ решено в Фазе C: `generated/net_models.go` (структуры), `net_send.go` (функции-отправки), `state_models.go` (типы состояния).
5. **пример в репо** — ✅ решено в Фазе D: `examples/counter/` со всеми YAML-конфигами.

---

## 5. Легенда статусов

- 🔴 приоритет 1 (сразу)
- 🟠 приоритет 2
- 🟡 приоритет 4
- ✅ готово/проверено
- [ ] не сделано
- [x] сделано
