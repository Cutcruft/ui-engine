package module

import (
	"fmt"
	"os"
	"path/filepath"
)

type ScaffoldOpts struct {
	Type        string
	TS          bool
	WithTests   bool
	WithDocs    bool
	WithExample bool
}

// Scaffold creates a new module template (legacy, without opts)
func Scaffold(targetDir, name, typeName string) error {
	return ScaffoldWithOpts(targetDir, name, ScaffoldOpts{Type: typeName})
}

// ScaffoldWithOpts creates a new module template with options
func ScaffoldWithOpts(targetDir, name string, opts ScaffoldOpts) error {
	typeName := opts.Type
	if typeName == "" {
		typeName = "component"
	}
	allowed := map[string]bool{"component": true, "layout": true, "wrapper": true, "service": true}
	if !allowed[typeName] {
		return fmt.Errorf("unknown module type %s (allowed: component, layout, wrapper, service)", typeName)
	}
	modDir := filepath.Join(targetDir, name)
	if _, err := os.Stat(modDir); err == nil {
		return fmt.Errorf("directory %s already exists", modDir)
	}
	if err := os.MkdirAll(modDir, 0755); err != nil {
		return err
	}
	// module.yaml
	entryJS := "js/bridge.js"
	if opts.TS {
		entryJS = "src/index.ts"
	}
	manifest := fmt.Sprintf(`module:
  name: %s
  version: 0.1.0
  type: %s
  description: "Модуль %s для ui-engine"
  author: ""
  license: MIT
  entry:
    js: %s
  components:
    - name: %s
      props:
        variant: {type: string, default: primary}
`, name, typeName, name, entryJS, name)
	if err := os.WriteFile(filepath.Join(modDir, "module.yaml"), []byte(manifest), 0644); err != nil {
		return err
	}
	// README
	readme := fmt.Sprintf("# %s\n\nМодуль `%s` типа `%s` для ui-engine.\n\n## Использование\n\n```yaml\n- component: %s\n  props:\n    variant: primary\n```\n\n## Разработка\n\n```sh\n# TS\nnpm run build\n```\n", name, name, typeName, name)
	os.WriteFile(filepath.Join(modDir, "README.md"), []byte(readme), 0644)
	if opts.TS {
		// src/index.ts
		srcDir := filepath.Join(modDir, "src")
		os.MkdirAll(srcDir, 0755)
		tsBridge := fmt.Sprintf(`import type { ComponentHandle } from "../../../runtime-js/src/types";

const %s: ComponentHandle = {
  mount(container, props, onEvent) {
    const el = document.createElement("div");
    el.textContent = "%s: " + JSON.stringify(props);
    el.style.padding = "8px";
    el.style.border = "1px dashed #ccc";
    container.appendChild(el);
    return {
      update(newProps) { el.textContent = "%s: " + JSON.stringify(newProps); },
      unmount() { el.remove(); }
    };
  }
};

export default %s;
(window as any).UIEngineModules = (window as any).UIEngineModules || {};
(window as any).UIEngineModules["%s"] = %s;
`, name, name, name, name, name, name)
		os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte(tsBridge), 0644)
		// tsconfig для модуля
		tscfg := `{
  "extends": "../../tsconfig.json",
  "compilerOptions": { "outDir": "./js", "rootDir": "./src" },
  "include": ["src/**/*"]
}`
		os.WriteFile(filepath.Join(modDir, "tsconfig.json"), []byte(tscfg), 0644)
		// package.json
		pkg := fmt.Sprintf(`{"name": "@ui-engine/%s","version":"0.1.0","type":"module","scripts":{"build":"tsc"}}`, name)
		os.WriteFile(filepath.Join(modDir, "package.json"), []byte(pkg), 0644)
	} else {
		// js/bridge.js
		jsDir := filepath.Join(modDir, "js")
		os.MkdirAll(jsDir, 0755)
		bridge := fmt.Sprintf(`// %s — JS bridge для ui-engine
// Экспортирует компонент %s
window.UIEngineModules = window.UIEngineModules || {};
window.UIEngineModules["%s"] = {
  mount(container, props, onEvent) {
    const el = document.createElement("div");
    el.textContent = "%s: " + JSON.stringify(props);
    el.style.padding = "8px";
    el.style.border = "1px dashed #ccc";
    container.appendChild(el);
    return { update(newProps) { el.textContent = "%s: " + JSON.stringify(newProps); }, unmount() { el.remove(); } };
  }
};
`, name, name, name, name, name)
		os.WriteFile(filepath.Join(jsDir, "bridge.js"), []byte(bridge), 0644)
	}
	// ui/component.yaml example
	uiDir := filepath.Join(modDir, "ui")
	os.MkdirAll(uiDir, 0755)
	compYAML := fmt.Sprintf(`# Пример компонента %s
component: %s
props:
  variant: primary
  size: md
`, name, name)
	os.WriteFile(filepath.Join(uiDir, "component.yaml"), []byte(compYAML), 0644)
	if opts.WithTests {
		testContent := fmt.Sprintf(`package %s_test

import "testing"

func Test%s(t *testing.T) {
  // TODO: picks
}
`, name, name)
		os.WriteFile(filepath.Join(modDir, name+"_test.go"), []byte(testContent), 0644)
		// JS test
		jsTest := fmt.Sprintf(`// %s — e2e
console.log("test %s");
`, name, name)
		os.WriteFile(filepath.Join(modDir, "test.js"), []byte(jsTest), 0644)
	}
	if opts.WithDocs {
		docsContent := "# " + name + "\n\nДокументация модуля " + name + ".\n\n## Props\n\n- variant — primary | secondary\n"
		os.MkdirAll(filepath.Join(modDir, "docs"), 0755)
		os.WriteFile(filepath.Join(modDir, "docs", "README.md"), []byte(docsContent), 0644)
	}
	if opts.WithExample {
		exDir := filepath.Join(modDir, "example")
		os.MkdirAll(exDir, 0755)
		exYAML := fmt.Sprintf(`screen: demo
name: demo
layout:
  component: %s
  props:
    variant: primary
`, name)
		os.WriteFile(filepath.Join(exDir, "demo.yaml"), []byte(exYAML), 0644)
	}

	fmt.Printf("✓ модуль %s (%s) создан в %s\n", name, typeName, modDir)
	if opts.TS {
		fmt.Printf("  TS: src/index.ts + tsconfig.json\n")
	}
	if opts.WithTests {
		fmt.Printf("  tests: %s_test.go, test.js\n", name)
	}
	return nil
}
