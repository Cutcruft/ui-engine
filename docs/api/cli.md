# CLI API

## init / create product

```sh
ui-engine init [dir] [--wizard]
ui-engine create product <name> [--template counter|todo|blank]
ui-engine product add <screen|state|module> <name> [--dir dir]
ui-engine product list [dir]
```

## build / dev / serve

```sh
ui-engine build [dir]   # -> bin/examples/<name>/{index.html,wasm/main.wasm}
ui-engine dev [dir]     # :8033 + hot-reload (YAML без пересборки, Go/JS с пересборкой)
ui-engine serve [dir]   # статика из bin/
```

## module

```sh
ui-engine module new <name> [--type component|layout|wrapper|service] [--ts] [--with-tests] [--with-docs] [--with-example] [--dir ./path]
ui-engine module add <name|path>[@version] [--local]
ui-engine module list [dir]
ui-engine module remove <name>
```

## lint / gen

```sh
ui-engine lint [dir]  # JSON-Schema + структура
ui-engine gen [dir]   # -> generated/{state_models.go,net_models.go,net_send.go}
```

## version / help

```sh
ui-engine version
ui-engine help
```
