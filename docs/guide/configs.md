# Конфиги

## app.yaml

```yaml
name: my-app
root: main
screensDir: screens
theme: theme.yaml
state: state.yaml
net: net.yaml
hooks: hooks.yaml
keys: keys.yaml
```

## state.yaml

```yaml
state:
  counter: {type: object, fields: {count: {type: int, default: 0}}}
  todos: {type: list, items: {type: object, fields: {id: {type: string}, title: {type: string}}}}
```

## theme.yaml

```yaml
active: light
tokens: {colors: {primary: "#6366f1"}, spacing: {md: "8px"}}
themes: {light: {app-bg: "#fff"}, dark: {app-bg: "#111"}}
```

## screens/*.yaml

```yaml
screen: main
layout:
  component: column
  children:
    - component: span
      text: "Привет {{state.user.name}}"
      animate: {type: fade, duration: fast}
```

## net.yaml / hooks.yaml / keys.yaml

См. `examples/counter`.
