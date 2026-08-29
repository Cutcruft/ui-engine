package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Product handles `ui-engine create product` and `ui-engine product ...`
func Product(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("использование: ui-engine product <add|list> [args] или ui-engine create product <name>")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "add":
		return productAdd(rest)
	case "list":
		return productList(rest)
	default:
		return fmt.Errorf("неизвестная подкоманда product %s (add|list)", sub)
	}
}

// CreateProduct handles `ui-engine create product <name> [--template counter|todo|blank]`
func CreateProduct(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("использование: ui-engine create product <name> [--template counter|todo|blank]")
	}
	name := args[0]
	template := "counter"
	for i := 1; i < len(args); i++ {
		if args[i] == "--template" && i+1 < len(args) {
			template = args[i+1]
			i++
		}
	}
	target := filepath.Join(".", name)
	if err := InitWithTemplate(target, template); err != nil {
		return err
	}
	fmt.Printf("✓ продукт %s создан (шаблон: %s) в %s\n", name, template, target)
	fmt.Printf("  cd %s && ../../bin/ui-engine dev\n", name)
	return nil
}

func productAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("использование: ui-engine product add <screen|state|module> <name> [--dir dir]")
	}
	typ := args[0]
	name := args[1]
	dir := "."
	for i := 2; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			dir = args[i+1]
			i++
		}
	}
	switch typ {
	case "screen":
		return addScreen(dir, name)
	case "state":
		return addState(dir, name)
	case "module":
		// делегируем в module add
		return fmt.Errorf("используйте: ui-engine module add <name>")
	default:
		return fmt.Errorf("неизвестный тип %s (screen|state|module)", typ)
	}
}

func productList(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	ls, err := newLayout(dir)
	if err != nil {
		return err
	}
	fmt.Printf("продукт: %s\n", ls.Root)
	// screens
	entries, _ := os.ReadDir(ls.ScreensDir)
	fmt.Printf("экраны (%d):\n", len(entries))
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			fmt.Printf("  • %s\n", e.Name())
		}
	}
	// modules
	modsDir := filepath.Join(ls.Root, "modules")
	if entries, err := os.ReadDir(modsDir); err == nil {
		fmt.Printf("модули (%d):\n", len(entries))
		for _, e := range entries {
			if e.IsDir() {
				fmt.Printf("  • %s\n", e.Name())
			}
		}
	}
	return nil
}

func addScreen(dir, name string) error {
	ls, err := newLayout(dir)
	if err != nil {
		return err
	}
	path := filepath.Join(ls.ScreensDir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("экран %s уже существует", name)
	}
	content := fmt.Sprintf(`screen: %s
name: %s
layout:
  component: div
  props:
    style: "padding:24px;"
  children:
    - component: span
      text: "Экран %s"
`, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("✓ экран %s создан: %s\n", name, path)
	return nil
}

func addState(dir, name string) error {
	// добавляет поле в state.yaml
	ls, err := newLayout(dir)
	if err != nil {
		return err
	}
	statePath := filepath.Join(ls.Root, "state.yaml")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return err
	}
	// простой append
	extra := fmt.Sprintf("\n  %s:\n    type: string\n    default: \"\"\n", name)
	// вставляем перед концом
	content := string(data)
	// naive: append at end of state
	content += extra
	return os.WriteFile(statePath, []byte(content), 0644)
}

// InitWithTemplate creates a project with a template
func InitWithTemplate(target, template string) error {
	// сначала обычный init
	if err := Init(target); err != nil {
		return err
	}
	// затем применяем шаблон
	switch template {
	case "todo":
		return applyTodoTemplate(target)
	case "blank":
		// уже blank
		return nil
	case "counter", "":
		// counter уже по умолчанию
		return nil
	default:
		return fmt.Errorf("неизвестный шаблон %s (counter|todo|blank)", template)
	}
}

func applyTodoTemplate(target string) error {
	// перезаписываем state.yaml и screens/main.yaml для todo
	stateContent := `state:
  todos:
    type: list
    items:
      type: object
      fields:
        id: {type: string, required: true}
        title: {type: string, required: true}
        done: {type: bool, default: false}
    default: []
  filter:
    type: string
    default: all
`
	screenContent := `screen: main
name: main
layout:
  component: div
  props:
    style: "padding:24px;max-width:800px;margin:0 auto;"
  children:
    - component: span
      text: "Todo — {{state.todos.length}} задач"
    - component: div
      repeat: state.todos
      children:
        - component: div
          props:
            style: "display:flex;gap:8px;padding:8px;border:1px solid #e5e7eb;border-radius:6px;"
          children:
            - component: span
              text: "{{item.title}}"
            - component: span
              text: "{{item.done}}"
`
	if err := os.WriteFile(filepath.Join(target, "state.yaml"), []byte(stateContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(target, "screens", "main.yaml"), []byte(screenContent), 0644); err != nil {
		return err
	}
	return nil
}

// InitWizard runs an interactive wizard for `ui-engine init`
func InitWizard(target string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Создание продукта ui-engine\n")
	fmt.Printf("Каталог [%s]: ", target)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name != "" {
		target = name
	}
	fmt.Printf("Шаблон (counter/todo/blank) [counter]: ")
	tmpl, _ := reader.ReadString('\n')
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		tmpl = "counter"
	}
	fmt.Printf("Тема (light/dark/ocean) [light]: ")
	theme, _ := reader.ReadString('\n')
	theme = strings.TrimSpace(theme)
	if theme == "" {
		theme = "light"
	}
	if err := InitWithTemplate(target, tmpl); err != nil {
		return err
	}
	// theme
	if theme != "light" {
		themePath := filepath.Join(target, "theme.yaml")
		data, _ := os.ReadFile(themePath)
		content := strings.Replace(string(data), "active: light", "active: "+theme, 1)
		os.WriteFile(themePath, []byte(content), 0644)
	}
	fmt.Printf("\n✓ продукт создан в %s\n", target)
	fmt.Printf("  cd %s && ../../bin/ui-engine dev\n", target)
	return nil
}
