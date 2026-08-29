package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ui-engine/cli/internal/schema"
	"github.com/ui-engine/core/config"
)

// Lint валидирует YAML-конфиги проекта.
func Lint(root string) error {
	ls, err := newLayout(root)
	if err != nil {
		return err
	}

	appPath := filepath.Join(ls.Root, "app.yaml")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return fmt.Errorf("app.yaml не найден в %s", ls.Root)
	}

	loader := config.NewLoader(ls.Root)
	app, err := loader.LoadApp()
	if err != nil {
		return fmt.Errorf("app.yaml: %w", err)
	}
	fmt.Printf("✓ app.yaml (%s), root=%s\n", app.Name, app.Root)

	// JSON-Schema: app.yaml.
	if data, err := os.ReadFile(appPath); err == nil {
		if err := schema.Validate("app", data); err != nil {
			return fmt.Errorf("app.yaml schema: %w", err)
		}
	}

	scDir := app.ScreensDir
	if !filepath.IsAbs(scDir) {
		scDir = filepath.Join(ls.Root, scDir)
	}
	screens, err := loader.LoadScreens(scDir)
	if err != nil {
		return fmt.Errorf("screens: %w", err)
	}
	fmt.Printf("✓ screens: %d найдено\n", len(screens))
	if _, ok := screens[app.Root]; !ok {
		return fmt.Errorf("root-экран '%s' не найден среди экранов (%v)", app.Root, screenNames(screens))
	}
	fmt.Printf("✓ root-экран '%s' существует\n", app.Root)

	// JSON-Schema: каждый экран.
	entries, _ := os.ReadDir(scDir)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(scDir, e.Name()))
		if err != nil {
			continue
		}
		if err := schema.Validate("screen", data); err != nil {
			return fmt.Errorf("screen %s schema: %w", e.Name(), err)
		}
	}
	fmt.Println("✓ screens schema")

	if app.ThemePath != "" {
		tp := app.ThemePath
		if !filepath.IsAbs(tp) {
			tp = filepath.Join(ls.Root, tp)
		}
		if _, err := loader.LoadTheme(tp); err != nil {
			return fmt.Errorf("theme: %w", err)
		}
		if data, err := os.ReadFile(tp); err == nil {
			if err := schema.Validate("theme", data); err != nil {
				return fmt.Errorf("theme schema: %w", err)
			}
		}
		fmt.Println("✓ theme.yaml")
	}

	// net/hooks/keys — опциональные, валидируем если есть.
	for _, item := range []struct{ name, path, def string }{
		{"net", app.NetPath, "net.yaml"},
		{"state", app.StatePath, "state.yaml"},
		{"hooks", app.HooksPath, "hooks.yaml"},
		{"keys", app.KeysPath, "keys.yaml"},
	} {
		p := filepath.Join(ls.Root, item.path)
		if item.path == "" {
			p = filepath.Join(ls.Root, item.def)
		}
		if data, err := os.ReadFile(p); err == nil {
			if err := schema.Validate(item.name, data); err != nil {
				return fmt.Errorf("%s schema: %w", item.name, err)
			}
			fmt.Printf("✓ %s.yaml\n", item.name)
		}
	}

	fmt.Println("Lint пройден без ошибок.")
	return nil
}

func screenNames(m map[string]*config.Screen) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
