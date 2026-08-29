# Анимации

Декларативно через `animate` на узле или экране.

## Типы

- `fade` — прозрачность
- `slide` — смещение (`direction: left/right/up/down`)
- `scale` — масштаб
- `spring` — пружина (`cubic-bezier(0.34,1.56,0.64,1)`)

## Использование

```yaml
screen: main
animate: {type: fade, duration: 300}
layout:
  children:
    - component: span
      if: state.show
      animate: {type: fade, duration: fast}  # токен из theme
      text: "Привет"

    - component: div
      repeat: state.todos
      animate: fade
      children:
        - component: div
          animate: {type: slide, duration: 200, direction: up}
          text: "{{item.title}}"
```

`duration`/`easing` могут быть токенами из `theme.yaml:animations`:

```yaml
tokens:
  animations:
    durationFast: "150ms"
    easingSpring: "cubic-bezier(0.34, 1.56, 0.64, 1)"
```

```yaml
animate: {type: spring, duration: durationFast, easing: easingSpring}
```

## Переходы экранов

```yaml
screen: about
animate: {type: slide, duration: 250, direction: right}
```

## Темизация переходов

```css
* { transition: background-color 0.3s ease, color 0.3s ease; }
```

Генерируется автоматически из `theme` (`core/theme/theme.go:48`).
