# Темизация

Дизайн-токены + кастомные темы.

## theme.yaml

```yaml
active: light
tokens:
  colors: {primary-600: "#3b82f6", neutral-100: "#f3f4f6"}
  spacing: {sm: "4px", md: "8px", lg: "16px"}
  radius: {sm: "4px", md: "8px"}
  typography: {fontSans: "Inter, sans-serif"}
  shadows: {sm: "0 1px 2px rgba(0,0,0,0.05)"}
  animations: {durationFast: "150ms", easingSpring: "cubic-bezier(...)"}
themes:
  light: {app-bg: "#fff", app-text: "#111827"}
  dark: {app-bg: "#111827", app-text: "#f9fafb"}
  ocean: {app-bg: "#e0f2fe", app-text: "#082f49"} # кастомная
```

Токены → CSS-переменные `--colors-primary-600`, `--spacing-md` и т.д.

## Переключение

```sh
# YAML
state:
  ui: {fields: {theme: {type: string, default: light}}}

# экран
- component: button
  on: {click: "set state.ui.theme = dark"}
```

```js
// JS
window.UIEngine.setState("state.ui.theme", "ocean")
window.UIEngine.getState("state.ui.theme")
```

Переключение анимируется (`* {transition: background-color 0.3s}`).

## Валидация и hot-reload

```sh
ui-engine lint  # проверяет theme.yaml
# правка theme.yaml → dev-сервер hot-reload + injectThemeStyle
```

## Генерация TS типов

Токены генерируются в `generated/theme.ts` (если нужен).
