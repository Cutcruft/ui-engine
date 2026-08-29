package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Build собирает WASM-ядро и статический бандл проекта в bin/.
func Build(root string) error {
	ls, err := newLayout(root)
	if err != nil {
		return err
	}
	if err := ensureDir(ls.Build); err != nil {
		return err
	}
	if err := ensureDir(filepath.Join(ls.Build, "wasm")); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(ls.Root, "app.yaml")); err != nil {
		return fmt.Errorf("не найден app.yaml в %s (запустите ui-engine init)", ls.Root)
	}

	// 1. Проверяем наличие wasm_exec.js (идёт с Go).
	wasmExec := goWasmExecPath()
	if wasmExec == "" {
		return fmt.Errorf("wasm_exec.js не найден в GOROOT %s", runtime.GOROOT())
	}
	if err := copyFile(wasmExec, filepath.Join(ls.Build, "wasm_exec.js")); err != nil {
		return err
	}

	// 2. Собираем ядро в wasm (кладём в bin/wasm/main.wasm).
	if err := buildWasm(ls.Root, filepath.Join(ls.Build, "wasm", "main.wasm")); err != nil {
		return err
	}

	// 3. Копируем пользовательский bootstrap JS/TS (или генерируем дефолтный).
	srcBootstrapJS := filepath.Join(ls.Root, "src", "js", "app.js")
	srcBootstrapTS := filepath.Join(ls.Root, "src", "js", "app.ts")
	if _, err := os.Stat(srcBootstrapTS); err == nil {
		// TS — берём скомпилированный из bin/ts (repo root)
		repoRoot := ""
		if sd := findStdlibDir(ls.Root); sd != "" {
			repoRoot = filepath.Dir(sd)
		} else {
			repoRoot = filepath.Join(ls.Root, "..", "..")
		}
		compiledTS := filepath.Join(repoRoot, "bin", "ts", "examples", "counter", "src", "js", "app.js")
		if _, err := os.Stat(compiledTS); err == nil {
			copyFile(compiledTS, filepath.Join(ls.Build, "app.js"))
		} else {
			// fallback — копируем TS как JS (если tsc не запускался, TS почти JS)
			copyFile(srcBootstrapTS, filepath.Join(ls.Build, "app.js"))
		}
	} else if _, err := os.Stat(srcBootstrapJS); err == nil {
		copyFile(srcBootstrapJS, filepath.Join(ls.Build, "app.js"))
	} else {
		if err := os.WriteFile(filepath.Join(ls.Build, "app.js"), []byte(bootstrapJS), 0o644); err != nil {
			return err
		}
	}
	// 3b. Копируем runtime-js (TS bridge)
	runtimeJS := findStdlibDir(ls.Root)
	if runtimeJS != "" {
		// runtime-js/src/index.ts -> bin/runtime.js
		compiledRuntime := filepath.Join(filepath.Dir(runtimeJS), "bin", "ts", "runtime-js", "src", "index.js")
		if _, err := os.Stat(compiledRuntime); err == nil {
			copyFile(compiledRuntime, filepath.Join(ls.Build, "runtime.js"))
		} else {
			// fallback: js/bridge.js
			if _, err := os.Stat(filepath.Join(runtimeJS, "index.js")); err == nil {
				copyFile(filepath.Join(runtimeJS, "index.js"), filepath.Join(ls.Build, "runtime.js"))
			}
		}
	}
	// Копируем модули (stdlib) JS/CSS если есть
	modulesDir := filepath.Join(ls.Root, "modules")
	if _, err := os.Stat(modulesDir); err == nil {
		entries, _ := os.ReadDir(modulesDir)
		for _, e := range entries {
			if e.IsDir() {
				modJS := filepath.Join(modulesDir, e.Name(), "js", "bridge.js")
				if _, err := os.Stat(modJS); err == nil {
					dst := filepath.Join(ls.Build, "modules", e.Name(), "bridge.js")
					ensureDir(filepath.Dir(dst))
					copyFile(modJS, dst)
				}
				modCSS := filepath.Join(modulesDir, e.Name(), "css")
				if _, err := os.Stat(modCSS); err == nil {
					copyDir(modCSS, filepath.Join(ls.Build, "modules", e.Name(), "css"))
				}
			}
		}
	}
	// также копируем stdlib модули напрямую если они используются без установки (для dev)
	stdlibDir := findStdlibDir(ls.Root)
	if stdlibDir != "" {
		for _, modName := range []string{"button", "layout", "richtext"} {
			modJS := filepath.Join(stdlibDir, modName, "js", "bridge.js")
			if _, err := os.Stat(modJS); err == nil {
				dst := filepath.Join(ls.Build, "modules", modName, "bridge.js")
				ensureDir(filepath.Dir(dst))
				if _, err := os.Stat(dst); os.IsNotExist(err) {
					copyFile(modJS, dst)
				}
			}
			// также копируем скомпилированный TS если есть (перезаписываем)
			compiledMod := filepath.Join(filepath.Dir(stdlibDir), "bin", "ts", "stdlib", modName, "src", "index.js")
			if _, err := os.Stat(compiledMod); err == nil {
				copyFile(compiledMod, filepath.Join(ls.Build, "modules", modName, "bridge.js"))
			}
		}
	}

	// 4. index.html.
	if err := os.WriteFile(filepath.Join(ls.Build, "index.html"), []byte(indexHTML), 0o644); err != nil {
		return err
	}

	fmt.Printf("Сборка завершена: %s\n", ls.Build)
	return nil
}

func buildWasm(root, out string) error {
	// Ядро в core/wasm. Используем go.work из корня репо.
	// Находим путь к пакету main ядра относительно проекта.
	coreDir := findCoreDir(root)
	if coreDir == "" {
		return fmt.Errorf("core/wasm не найден (ожидается рядом с корнем репо)")
	}

	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = coreDir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка сборки wasm: %v\n%s", err, string(outBytes))
	}
	return nil
}

// findCoreDir ищет core/wasm следующими путями:
// 1. env UI_CORE_DIR
// 2. относительно бинаря CLI (например bin/ui-engine -> ../core/wasm)
// 3. рядом с проектом: ../core/wasm (examples/* внутри репо)
func findCoreDir(root string) string {
	if env := os.Getenv("UI_CORE_DIR"); env != "" {
		if fi, err := os.Stat(filepath.Join(env, "main.go")); err == nil && !fi.IsDir() {
			return env
		}
		if fi, err := os.Stat(env); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(env, "main.go")); err == nil {
				return env
			}
		}
	}

	// Относительно бинаря CLI.
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		cand := filepath.Join(exeDir, "..", "core", "wasm")
		if _, err := os.Stat(filepath.Join(cand, "main.go")); err == nil {
			return filepath.Clean(cand)
		}
	}

	candidates := []string{
		filepath.Join(root, "core", "wasm"),
		filepath.Join(root, "..", "core", "wasm"),
		filepath.Join(root, "..", "core"),
	}
	for _, c := range candidates {
		abs := filepath.Clean(c)
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(abs, "main.go")); err == nil {
				return abs
			}
		}
	}
	return ""
}

func goWasmExecPath() string {
	goroot := runtime.GOROOT()
	p := filepath.Join(goroot, "lib", "wasm", "wasm_exec.js")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := ensureDir(dst); err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func findStdlibDir(root string) string {
	cands := []string{
		filepath.Join(root, "..", "stdlib"),
		filepath.Join(root, "stdlib"),
		filepath.Join(filepath.Dir(root), "stdlib"),
	}
	if exe, err := os.Executable(); err == nil {
		cands = append(cands, filepath.Join(filepath.Dir(exe), "..", "stdlib"))
		cands = append(cands, filepath.Join(filepath.Dir(exe), "..", "..", "stdlib"))
	}
	for _, c := range cands {
		if fi, err := os.Stat(filepath.Join(c, "button", "module.yaml")); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

const indexHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <link rel="icon" href="data:," />
  <title>ui-engine app</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.20.1/cdn/themes/light.css" />
  <script type="module" src="https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.20.1/cdn/shoelace-autoloader.js"></script>
  <style>html,body{margin:0;height:100%;}</style>
</head>
<body>
  <div id="root"></div>
  <script src="wasm_exec.js"></script>
  <script type="module" src="runtime.js"></script>
  <script type="module" src="modules/button/bridge.js"></script>
  <script type="module" src="modules/layout/bridge.js"></script>
  <script type="module" src="modules/richtext/bridge.js"></script>
  <script type="module" src="app.js"></script>
</body>
</html>
`
