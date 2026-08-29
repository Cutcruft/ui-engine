# Contributing

## Разработка

```sh
# зависимости
npm install
task build-cli  # -> bin/ui-engine

# тесты
task test
task lint
task all  # полная проверка

# dev
bin/ui-engine dev examples/counter
# http://127.0.0.1:8033

# доки
npm run docs:dev    # http://localhost:5173
npm run docs:build  # -> docs/.vitepress/dist
```

## Структура

- `core/` — Go-ядро (config, state, vdom, diff, dom, runtime, wasm, theme) → `bin/core.wasm`
- `cli/` — CLI (`init`/`build`/`dev`/`module`/`product`)
- `runtime-js/` — TS-мост `window.UIEngine` (`src/index.ts` → `bin/ts`)
- `stdlib/` — внешние модули (`button` PrimeVue, `layout` Flex, `richtext` Tiptap) — `src/index.ts` → `js/bridge.js`
- `examples/` — примеры (`counter`, `todo` шаблон)
- `docs/` — VitePress

## Модули

```sh
bin/ui-engine module new my-mod --type component --ts --with-tests
bin/ui-engine module add my-mod --local ./my-mod
```

## Коммиты

- `feat: ...`, `fix: ...`, `docs: ...`, `chore: ...`
- PR → `task all` зелёный, `e2e` PASS

## Релиз

```sh
# version bump + tag + GitHub Release с bin/ui-engine-* 
# + npm publish (@ui-engine/cli)
# + install.sh (https://ui-engine.dev/install.sh)
```
