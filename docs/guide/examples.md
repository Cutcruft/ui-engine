# Примеры

## counter

`examples/counter` — полный демо: счётчик + `if` + `sl-alert` + `repeat: state.todos` + `sl-input` (`bind`/`$event`) + `richtext` + `JS Bridge` + `ocean` тема + `animate`.

```sh
ui-engine dev examples/counter
# http://127.0.0.1:8033
```

## todo (шаблон)

```sh
ui-engine create product my-todo --template todo
# -> state: todos (list), filter, screens/main.yaml с repeat
```

## Использование модулей

```sh
ui-engine module add button
ui-engine module add layout
ui-engine module add richtext
```
