# Быстрый старт

## 1. Создать продукт

```sh
# интерактивный wizard
ui-engine init --wizard
# или
ui-engine create product my-app --template counter
# шаблоны: counter | todo | blank
cd my-app
```

Структура:
```
my-app/
├── app.yaml
├── theme.yaml
├── state.yaml
├── screens/main.yaml
├── screens/about.yaml
└── src/js/app.ts
```

## 2. Dev-сервер

```sh
ui-engine dev
# http://127.0.0.1:8033
# hot-reload: правка YAML -> живой DOM без пересборки
```

## 3. Первый экран

`app.yaml`:
```yaml
name: my-app
root: main
screensDir: screens
```

`screens/main.yaml`:
```yaml
screen: main
name: main
animate: {type: fade, duration: fast}
layout:
  component: column
  children:
    - component: span
      text: "Привет, {{state.user.name}}"
      animate: fade
    - component: button
      props: {variant: primary, label: "+1"}
      on: {click: "inc state.counter.count"}
```

`state.yaml`:
```yaml
state:
  counter: {type: object, fields: {count: {type: int, default: 0}}}
  user: {type: object, fields: {name: {type: string, default: "Гость"}}}
```

## 4. Сборка

```sh
ui-engine build
# -> bin/examples/my-app/{index.html, wasm/main.wasm, app.js}
ui-engine serve
# http://127.0.0.1:8033
```

## 5. Добавить экран

```sh
ui-engine product add screen profile
# -> screens/profile.yaml
```

## 6. Добавить модуль

```sh
ui-engine module add button
ui-engine module add layout
ui-engine module add richtext
```
