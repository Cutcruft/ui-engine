package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// Init создаёт новый проект в target dir: скелет папок и YAML-конфиги.
func Init(target string) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := ensureDir(abs); err != nil {
		return err
	}
	ls, err := newLayout(abs)
	if err != nil {
		return err
	}

	dirs := []string{
		ls.ScreensDir,
		filepath.Join(abs, "assets"),
	}
	for _, d := range dirs {
		if err := ensureDir(d); err != nil {
			return err
		}
	}

	files := map[string]string{
		"app.yaml":           appYAML,
		"theme.yaml":         themeYAML,
		"screens/main.yaml":  screenMainYAML,
		"screens/about.yaml": screenAboutYAML,
		".gitignore":         gitignore,
	}

	for name, content := range files {
		path := filepath.Join(abs, name)
		if _, err := os.Stat(path); err == nil {
			continue // не перезаписываем существующие
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	fmt.Printf("Проект создан: %s\n", abs)
	fmt.Println("Дальше: ui-engine build && ui-engine dev")
	return nil
}

const appYAML = `# Корневой конфиг приложения — только YAML, без JS.
name: my-app
root: main            # стартовый экран
entry: root           # id DOM-контейнера, куда монтируется ядро
screensDir: screens
theme: theme.yaml
themeActive: light
`

const themeYAML = `# Темы: имя -> css-переменные (потребляются Shoelace и кастомными стилями).
active: light
themes:
  light:
    sl-color-primary-600: "#3b82f6"
    sl-color-primary-500: "#60a5fa"
    sl-color-neutral-100: "#f3f4f6"
    sl-color-neutral-800: "#1f2937"
    sl-font-sans: "Inter, system-ui, sans-serif"
    app-bg: "#ffffff"
    app-text: "#111827"
  dark:
    sl-color-primary-600: "#3b82f6"
    sl-color-primary-500: "#60a5fa"
    sl-color-neutral-100: "#1f2937"
    sl-color-neutral-800: "#f3f4f6"
    sl-font-sans: "Inter, system-ui, sans-serif"
    app-bg: "#111827"
    app-text: "#f9fafb"
`

const screenMainYAML = `# Экран main. Дерево макета.
screen: main
name: main
layout:
  component: div
  key: root
  props:
    style: "display:flex;flex-direction:column;gap:24px;padding:24px;min-height:100vh;background:var(--app-bg);color:var(--app-text);font-family:var(--sl-font-sans);"
  children:
    - component: sl-card
      key: hero
      children:
        - component: div
          key: hero-body
          children:
            - component: sl-button
              key: btn
              label: "Нажми меня"
              on:
                click: action.toast
    - component: div
      key: statusbar
      props:
        style: "font-size:14px;color:var(--sl-color-neutral-400);"
      children:
        - component: span
          key: counter
          text: "Счётчик: {{state.counter.count}}"
`

const screenAboutYAML = `screen: about
name: about
layout:
  component: div
  key: root
  props:
    style: "padding:24px;"
  children:
    - component: span
      key: title
      text: "О проекте ui-engine"
`

const gitignore = `bin/
build/
dist/
*.wasm
*.js.map
coverage/
.DS_Store
`

const bootstrapJS = `// Bootstrap: загружает wasm-ядро и кладёт YAML-конфиги в window.__UI_CONFIG__.
// Здесь мы условно считаем, что конфиги встроены сборщиком в __BOOT_CONFIG__.
async function loadConfigs() {
  // В dev-режиме конфиги подаются с сервера; в prod — запекаются в бандл.
  const resp = await fetch('/__config__');
  if (!resp.ok) throw new Error('config fetch failed');
  return resp.json();
}

async function main() {
  const cfg = await loadConfigs();
  window.__UI_CONFIG__ = cfg;

  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(
    fetch('/wasm/main.wasm'),
    go.importObject,
  );
  go.run(result.instance);
  // Ядро вызвало __uiEngineReady при boot; держим ссылку чтобы не GC.
  window.__go = go;
}

main().catch((e) => {
  console.error('ui-engine bootstrap error', e);
});
`

func init() { /* placeholder doc */ }
