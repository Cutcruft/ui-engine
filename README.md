# ui-engine

Движок для построения веб-интерфейсов по принципу «Go + WASM + YAML»: макет, темы, состояние, сети и клавиатура — декларативные YAML-конфиги, рантайм на Go в браузере (WASM).

## Как это работает

- **Конфиги** (`app.yaml`, `screens/*.yaml`, `theme.yaml`, `state.yaml`, `net.yaml`, `hooks.yaml`, `keys.yaml`) описывают всё приложение.
- **Ядро** (`core/`) — парсинг конфигов, реактивный `state.Store`, VDOM + diff, декларативный WS-клиент, мини-DSL действий, браузерный DOM-рендерер. Компилируется в WASM (`core/wasm`).
- **CLI** (`cli/`) — `init`, `build`, `dev` (hot-reload), `lint` (JSON-Schema валидация), `gen` (кодогенерация из контрактов), `serve`.
- **Реактивность**: поддеревья подписываются на пути состояния — мутация `state.<path>` пере-рендерит только связанную ветку (точечный diff), а не всё дерево.

## Быстрый старт

```sh
# 1. Собрать CLI
task build-cli            # -> bin/ui-engine

# 2. Создать проект
./bin/ui-engine init my-app

# 3. Собрать WASM-ядро и статический бандл
./bin/ui-engine build my-app

# 4. Dev-сервер с hot-reload (правка YAML -> живой DOM)
./bin/ui-engine dev my-app
```

Через `task` или `make`:

```sh
task test         # тесты core + cli
task build-cli    # CLI в bin/ui-engine
task wasm         # ядро в bin/core.wasm
make build        # тоже самое через Makefile
make test
make lint
make clean        # удалить bin/build/dist
make dev          # dev-сервер примера
make cover        # покрытие
task gen          # сгенерировать типы из net.yaml/state.yaml примера
task lint         # валидировать конфиги примера
task demo-dev     # dev-сервер примера (examples/counter)
task all          # полная проверка
```

**WebSocket — надёжность:** `core/net/client.go:42` — экспоненциальный реконнект (`BaseMS`/`MaxMS`/`Attempts`), `ping/pong` с `IntervalMS`/`TimeoutMS` и `lastPong`, `auth` refresh при `auth_required`, очередь `queue` при оффлайне + `flushQueue` на `onOpen`.

**Скрипты сборки:** `scripts/pull-build.sh [version] [--to bin]` — тянет `bin/ui-engine-*`, `core.wasm`, `*.uimod` с `https://github.com/ui-engine/ui-engine/releases`.

## Структура проекта

```
core/        — ядро: config, state, vdom, diff, dom, runtime, net, theme, wasm
cli/         — CLI: init/build/dev/lint/gen/serve/version
examples/counter/ — рабочий пример со всеми YAML-конфигами
docs/        — документация
stdlib/      — подключаемые стайл-системы и компоненты (например Shoelace)
tools/       — вспомогательные утилиты генерации
```

- `go.work` объединяет модули `core` и `cli`.
- Сгенерированный код — в папке `generated/` проекта (не редактировать вручную).

## Команды CLI

| Команда              | Описание |
|----------------------|----------|
| `init <dir>`         | создать новый проект |
| `build [dir]`        | собрать wasm-ядро и статический бандл |
| `dev [dir]`          | dev-сервер с hot-reload |
| `lint [dir]`         | валидировать YAML-конфиги (JSON-Schema + структура) |
| `gen [dir]`          | сгенерировать Go-типы из net.yaml/state.yaml в `generated/` |
| `serve [dir]`        | статический сервер собранного SPA |
| `version`            | показать версию |

## Документация

- [VitePress (красиво)](https://ui-engine.dev) — `npm run docs:dev` → http://localhost:5173
- [DESIGN.md](DESIGN.md) — полный дизайн движка.
- [TODO.md](TODO.md) — план разработки и статусы фаз.
- [CONTRIBUTING.md](CONTRIBUTING.md) — как контрибьютить.

## Установка

```sh
curl -fsSL https://ui-engine.dev/install.sh | sh
# или
npm i -g @ui-engine/cli
# или
go install github.com/ui-engine/cli@latest
```