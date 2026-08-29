// Command ui-engine — консольное приложение для разработки и управления
// проектами на движке ui-engine.
//
// Команды:
//
//	init <dir>     — создать новый проект (скелет папок + YAML конфиги)
//	build          — собрать WASM-ядро и статический бандл
//	dev            — dev-сервер с hot-reload (fsnotify: YAML + пересборка wasm)
//	lint           — валидация YAML конфигов
//	serve          — статический сервер собранного SPA
//	version        — показать версию
package main

import (
	"fmt"
	"os"

	"github.com/ui-engine/cli/internal/cmd"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	switch args[0] {
	case "init":
		// wizard если --wizard или интерактивно без аргументов
		if len(args) > 1 && args[1] == "--wizard" {
			target := "."
			if len(args) > 2 {
				target = args[2]
			}
			if err := cmd.InitWizard(target); err != nil {
				fail(err)
			}
		} else {
			target := "."
			if len(args) > 1 {
				target = args[1]
			}
			// если init без аргументов и stdin — tty, запускаем wizard
			if len(args) == 1 {
				if fi, _ := os.Stdin.Stat(); fi.Mode()&os.ModeCharDevice != 0 {
					// interactive
					if err := cmd.InitWizard(target); err != nil {
						fail(err)
					}
					break
				}
			}
			if err := cmd.Init(target); err != nil {
				fail(err)
			}
		}
	case "create":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "использование: ui-engine create product <name> [--template counter|todo|blank]")
			os.Exit(1)
		}
		switch args[1] {
		case "product":
			if err := cmd.CreateProduct(args[2:]); err != nil {
				fail(err)
			}
		default:
			fmt.Fprintf(os.Stderr, "неизвестная подкоманда create %s\n", args[1])
			os.Exit(1)
		}
	case "product":
		if err := cmd.Product(args[1:]); err != nil {
			fail(err)
		}
	case "build":
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		if err := cmd.Build(dir); err != nil {
			fail(err)
		}
	case "dev":
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		if err := cmd.Dev(dir); err != nil {
			fail(err)
		}
	case "lint":
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		if err := cmd.Lint(dir); err != nil {
			fail(err)
		}
	case "gen":
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		if err := cmd.Gen(dir); err != nil {
			fail(err)
		}
	case "serve":
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		if err := cmd.Serve(dir); err != nil {
			fail(err)
		}
	case "module":
		if err := cmd.Module(args[1:]); err != nil {
			fail(err)
		}
	case "version":
		fmt.Println("ui-engine 0.1.0")
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`ui-engine — CLI для движка построения веб-интерфейсов (Go + WASM + YAML)

  init [dir] [--wizard]          создать новый проект (wizard если tty)
  create product <name> [--template counter|todo|blank]  создать продукт
  product add <screen|state|module> <n>  добавить к продукту
  product list [dir]             список экранов/модулей продукта
  build [dir]  собрать wasm-ядро и статический бандл
  dev [dir]    dev-сервер с hot-reload
  lint [dir]   валидировать YAML конфиги
  gen [dir]    сгенерировать Go-типы из net.yaml/state.yaml в generated/
  serve [dir]  статический сервер для собранного SPA
  module new <name> [--type type] [--ts] [--with-tests] [--with-docs]  создать модуль
  module add <name|path> [--local]    подключить модуль
  module list                         список модулей
  module remove <name>                удалить модуль
  version      показать версию
  help         показать справку
`)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ui-engine:", err)
	os.Exit(1)
}
