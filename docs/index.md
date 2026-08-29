# ui-engine

<div class="hero">

**Go + WASM + YAML** — декларативные веб-интерфейсы без бэкенда в браузере.

Мощный движок для создания современных веб-приложений с помощью простых YAML-конфигов.

</div>

## ✨ Особенности

- **Декларативно** — макет, состояние, темы, сеть — всё в YAML
- **Реактивно** — точечный ре-рендер поддеревьев по подпискам, keyed-списки
- **Модульно** — внешние плагины через `ui-engine module add`, PrimeVue + Flex + Tiptap
- **Анимировано** — `animate: {type: fade, duration: fast}` + spring, токены из темы
- **Типизировано** — весь JS → TS, `window.UIEngine` с типами

## 🚀 Быстрый старт

```sh
# установка
curl -fsSL https://ui-engine.dev/install.sh | sh
# или
npm i -g @ui-engine/cli
# или
go install github.com/ui-engine/cli@latest

# создать продукт
ui-engine create product my-app --template counter
cd my-app
ui-engine dev
```

Откройте http://127.0.0.1:8033

## 📚 Документация

- [Установка](/guide/installation) — все способы установки
- [Быстрый старт](/guide/quickstart) — первый экран за 5 минут
- [Конфиги](/guide/configs) — app/state/theme/screens
- [Компоненты](/guide/components) — button, input, list, richtext
- [Модули](/guide/modules) — создание и подключение внешних модулей
- [Анимации](/guide/animations) — переходы, enter/leave, spring

## 🎨 Пример

```yaml
# screens/main.yaml
layout:
  component: column
  children:
    - component: span
      text: "Счётчик: {{state.counter.count}}"
      animate: {type: fade, duration: fast}
    - component: button
      props: {variant: primary}
      on: {click: "inc state.counter.count"}
      label: "+1"
```

## 📦 Модули

```sh
ui-engine module add button      # PrimeVue кнопки
ui-engine module add layout      # Vue Flex
ui-engine module add richtext    # Tiptap notion-like
```

<style>
.hero { text-align: center; padding: 48px 0; }
.hero h1 { font-size: 48px; font-weight: 800; background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
</style>
