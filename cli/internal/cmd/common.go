// Package cmd реализует команды CLI.
package cmd

import (
	"os"
	"path/filepath"
)

// Layout описывает структуру проекта (корень).
type Layout struct {
	Root       string // корень проекта (где app.yaml)
	Build      string // выходная папка собранного SPA
	ScreensDir string
}

func newLayout(root string) (*Layout, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// Build всегда в bin/ репо (bin/examples/<name> для примеров, bin/ для корня)
	buildDir := filepath.Join(abs, "bin")
	// если это пример (examples/*), кладём в repo/bin/examples/<name>
	if isExampleRoot(abs) {
		if repoRoot := findRepoRoot(abs); repoRoot != "" {
			buildDir = filepath.Join(repoRoot, "bin", "examples", filepath.Base(abs))
		}
	}
	return &Layout{
		Root:       abs,
		Build:      buildDir,
		ScreensDir: filepath.Join(abs, "screens"),
	}, nil
}

func isExampleRoot(p string) bool {
	// пример — папка с app.yaml и screens/
	if _, err := os.Stat(filepath.Join(p, "app.yaml")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(p, "screens")); err != nil {
		return false
	}
	// и находится под examples/
	return filepath.Base(filepath.Dir(p)) == "examples" || filepath.Base(p) == "counter" || filepath.Base(p) == "todo"
}

func findRepoRoot(start string) string {
	cur := start
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(cur, "go.work")); err == nil {
			return cur
		}
		if _, err := os.Stat(filepath.Join(cur, "Taskfile.yml")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// ensureDir создаёт директории, если их нет.
func ensureDir(p string) error { return os.MkdirAll(p, 0o755) }
