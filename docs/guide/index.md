# Введение

Добро пожаловать в **ui-engine** — движок для построения веб-интерфейсов по принципу **Go + WASM + YAML**.

## Что это?

- **Декларативно** — весь UI в YAML: `app.yaml`, `screens/*.yaml`, `state.yaml`, `theme.yaml`, `net.yaml`, `hooks.yaml`, `keys.yaml`.
- **Реактивно** — `state.Store` с подписками, точечный `diff`/`Patch` только грязных поддеревьев.
- **Модульно** — внешние плагины (`button` PrimeVue, `layout` Flex, `richtext` Tiptap) через `ui-engine module add`, строгая изоляция ядра (`window.UIEngine.registerComponent`).
- **Анимировано** — `animate: {type: fade, duration: fast}` + `spring`, токены из `theme.yaml`.
- **Типизировано** — весь JS → TS (`runtime-js/src`, `stdlib/*/src`), `window.UIEngine` с типами.

## Структура

- [Установка](/guide/installation) — `curl -fsSL https://ui-engine.dev/install.sh | sh` / `npm i -g @ui-engine/cli`
- [Быстрый старт](/guide/quickstart) — `ui-engine create product my-app --template counter` + `ui-engine dev`
- [Конфиги](/guide/configs) — `app`/`state`/`theme`/`screens`/`net`/`hooks`/`keys`
- [Компоненты](/guide/components) — `Button`/`InputText`/`Card`/`richtext`
- [Модули](/guide/modules) — создание и подключение внешних модулей
- [Анимации](/guide/animations) — `fade`/`slide`/`scale`/`spring`
- [Темизация](/guide/theming) — дизайн-токены + кастомные темы (`ocean`)
- [JS Bridge](/guide/js-bridge) — `window.UIEngine.dispatch`/`getState`/`setState`
- [Примеры](/guide/examples) — `counter` (счётчик + список + форма)

## Далее

- [Быстрый старт](/guide/quickstart) — первый экран за 5 минут
- [CLI API](/api/cli) — полный список команд
