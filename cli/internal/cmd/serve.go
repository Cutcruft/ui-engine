package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ui-engine/core/config"
	"gopkg.in/yaml.v3"
)

// Serve запускает статический сервер собранного приложения (bin/).
func Serve(root string) error {
	ls, err := newLayout(root)
	if err != nil {
		return err
	}
	addr := envOr("UI_ADDR", "127.0.0.1:8033")

	mux := http.NewServeMux()
	// отдаём собранные статические файлы
	fs := http.FileServer(http.Dir(ls.Build))
	mux.Handle("/", fs)
	// конфиги для bootstrap
	mux.HandleFunc("/__config__", func(w http.ResponseWriter, r *http.Request) {
		serveConfig(w, ls)
	})

	fmt.Printf("Сервер запущен: http://%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

// serveConfig отдаёт конфиги приложения как JSON: {app: stringYAML, theme: stringYAML, screens: {name: stringYAML}}.
func serveConfig(w http.ResponseWriter, ls *Layout) {
	appPath := filepath.Join(ls.Root, "app.yaml")
	appData, err := os.ReadFile(appPath)
	if err != nil {
		http.Error(w, "app.yaml not found", http.StatusNotFound)
		return
	}
	var app config.App
	if err := yaml.Unmarshal(appData, &app); err != nil {
		http.Error(w, "bad app.yaml: "+err.Error(), http.StatusBadRequest)
		return
	}

	themePath := app.ThemePath
	if !filepath.IsAbs(themePath) {
		themePath = filepath.Join(ls.Root, themePath)
	}
	themeData, _ := os.ReadFile(themePath)

	screens := map[string]string{}
	scDir := app.ScreensDir
	if !filepath.IsAbs(scDir) {
		scDir = filepath.Join(ls.Root, scDir)
	}
	if entries, err := os.ReadDir(scDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			data, _ := os.ReadFile(filepath.Join(scDir, e.Name()))
			name := trimYAMLExt(e.Name())
			screens[name] = string(data)
		}
	}

	payload := map[string]any{
		"app":     string(appData),
		"theme":   string(themeData),
		"screens": screens,
		"net":     readOptionalYAML(ls.Root, app.NetPath, "net.yaml"),
		"state":   readOptionalYAML(ls.Root, app.StatePath, "state.yaml"),
		"hooks":   readOptionalYAML(ls.Root, app.HooksPath, "hooks.yaml"),
		"keys":    readOptionalYAML(ls.Root, app.KeysPath, "keys.yaml"),
	}
	out := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	if err := out.Encode(payload); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func trimYAMLExt(name string) string { return name[:len(name)-len(filepath.Ext(name))] }

// readOptionalYAML читает файл конфига (путь из app.yaml или дефолтное имя),
// возвращая пустую строку, если файл отсутствует.
func readOptionalYAML(root, path, def string) string {
	if path == "" {
		path = def
	}
	full := filepath.Join(root, path)
	data, err := os.ReadFile(full)
	if err != nil {
		return ""
	}
	return string(data)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
