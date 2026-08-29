package module

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ui-engine/cli/internal/schema"
	"gopkg.in/yaml.v3"
)

// ProjectModulesDir is where modules are installed in a project
const ProjectModulesDir = "modules"

// Add installs a module into project
// source can be: "button", "button@1.0.0", "./local/path", "/abs/path", "https://..."
func Add(projectRoot, source string) error {
	var srcDir string
	var manifest *Manifest
	var version string

	// detect local path
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "../") {
		srcDir = source
		if !filepath.IsAbs(srcDir) {
			srcDir = filepath.Join(projectRoot, srcDir)
		}
		m, err := LoadManifest(srcDir)
		if err != nil {
			return err
		}
		manifest = m
	} else if strings.Contains(source, "@") {
		parts := strings.SplitN(source, "@", 2)
		name := parts[0]
		version = parts[1]
		srcDir = FindInStdlib(name)
		if srcDir == "" {
			return fmt.Errorf("module %s not found in stdlib (registry not yet implemented)", name)
		}
		m, err := LoadManifest(srcDir)
		if err != nil {
			return err
		}
		manifest = m
		if version != "" {
			manifest.Module.Version = version
		}
	} else {
		// name without version, e.g. "button"
		srcDir = FindInStdlib(source)
		if srcDir == "" {
			return fmt.Errorf("module %s not found in stdlib", source)
		}
		m, err := LoadManifest(srcDir)
		if err != nil {
			return err
		}
		manifest = m
	}

	// destination
	destDir := filepath.Join(projectRoot, ProjectModulesDir, manifest.Module.Name)
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("module %s already installed (use remove first)", manifest.Module.Name)
	}

	if err := copyDir(srcDir, destDir); err != nil {
		return fmt.Errorf("copy module: %w", err)
	}

	// update app.yaml
	if err := addToAppYAML(projectRoot, manifest.Module.Name, manifest.Module.Version, destDir); err != nil {
		// rollback
		os.RemoveAll(destDir)
		return err
	}

	fmt.Printf("✓ модуль %s@%s установлен в %s\n", manifest.Module.Name, manifest.Module.Version, destDir)
	return nil
}

// Remove uninstalls a module
func Remove(projectRoot, name string) error {
	destDir := filepath.Join(projectRoot, ProjectModulesDir, name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("module %s not installed", name)
	}
	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	// remove from app.yaml
	if err := removeFromAppYAML(projectRoot, name); err != nil {
		return err
	}
	fmt.Printf("✓ модуль %s удалён\n", name)
	return nil
}

// List shows installed modules
func List(projectRoot string) error {
	modsDir := filepath.Join(projectRoot, ProjectModulesDir)
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("модули не установлены")
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		fmt.Println("модули не установлены")
		return nil
	}
	fmt.Println("установленные модули:")
	for _, e := range entries {
		if e.IsDir() {
			m, err := LoadManifest(filepath.Join(modsDir, e.Name()))
			ver := "?"
			if err == nil {
				ver = m.Module.Version
			}
			fmt.Printf("  • %s@%s\n", e.Name(), ver)
		}
	}
	// also check app.yaml
	return nil
}

// FindInStdlib searches for module in stdlib
func FindInStdlib(name string) string {
	// try relative to project root's stdlib, and relative to cli's stdlib
	candidates := []string{
		filepath.Join("stdlib", name),
		filepath.Join("../stdlib", name),
		filepath.Join("../../stdlib", name),
		// absolute from repo root if projectRoot is known? We'll search via env
	}
	// also try from executable's dir
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "stdlib", name))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "..", "stdlib", name))
	}
	// try from current workdir's stdlib
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "stdlib", name))
		candidates = append(candidates, filepath.Join(wd, "..", "stdlib", name))
	}
	// also try via UI_ENGINE_ROOT env
	if root := os.Getenv("UI_ENGINE_ROOT"); root != "" {
		candidates = append(candidates, filepath.Join(root, "stdlib", name))
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "module.yaml")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func addToAppYAML(projectRoot, name, version, path string) error {
	appPath := filepath.Join(projectRoot, "app.yaml")
	data, err := os.ReadFile(appPath)
	if err != nil {
		return err
	}
	var app map[string]any
	if err := yaml.Unmarshal(data, &app); err != nil {
		return err
	}
	mods, _ := app["modules"].([]any)
	// check duplicate
	for _, m := range mods {
		if mm, ok := m.(map[string]any); ok {
			if mm["name"] == name {
				return fmt.Errorf("module %s already in app.yaml", name)
			}
		}
	}
	mods = append(mods, map[string]any{
		"name":    name,
		"version": version,
		"path":    strings.TrimPrefix(path, projectRoot+"/"),
	})
	app["modules"] = mods
	// validate via schema if available
	_ = schema.Validate // keep import
	newData, err := yaml.Marshal(app)
	if err != nil {
		return err
	}
	return os.WriteFile(appPath, newData, 0644)
}

func removeFromAppYAML(projectRoot, name string) error {
	appPath := filepath.Join(projectRoot, "app.yaml")
	data, err := os.ReadFile(appPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var app map[string]any
	if err := yaml.Unmarshal(data, &app); err != nil {
		return err
	}
	mods, _ := app["modules"].([]any)
	newMods := []any{}
	for _, m := range mods {
		if mm, ok := m.(map[string]any); ok {
			if mm["name"] == name {
				continue
			}
		}
		newMods = append(newMods, m)
	}
	if len(newMods) == 0 {
		delete(app, "modules")
	} else {
		app["modules"] = newMods
	}
	newData, err := yaml.Marshal(app)
	if err != nil {
		return err
	}
	return os.WriteFile(appPath, newData, 0644)
}
