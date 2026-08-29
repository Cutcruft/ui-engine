// Package config загружает и валидирует YAML-конфиги приложения:
// app.yaml, screens/*.yaml, theme.yaml.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// App — корневой конфиг приложения (app.yaml).
type App struct {
	Name        string   `yaml:"name"`
	Root        string   `yaml:"root"`          // стартовый экран/роут
	Modules     []Module `yaml:"modules"`       // подключаемые модули
	ThemePath   string   `yaml:"theme"`         // путь к theme.yaml
	ThemeActive string   `yaml:"themeActive"`   // активная тема (light/dark)
	ScreensDir  string   `yaml:"screensDir"`    // папка с экранами
	NetPath     string   `yaml:"net"`           // путь к net.yaml (опционально)
	StatePath   string   `yaml:"state"`         // путь к state.yaml (опционально)
	HooksPath   string   `yaml:"hooks"`         // путь к hooks.yaml
	KeysPath    string   `yaml:"keys"`          // путь к keys.yaml
	Entry       string   `yaml:"entry"`         // точка монтирования DOM (селектор)
}

// Module — описание подключения модуля.
type Module struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Path    string `yaml:"path"` // локальный путь (для файловых модулей)
}

// AnimateConfig — декларативная анимация узла/экрана
type AnimateConfig struct {
	Type      string `yaml:"type"`      // fade | slide | scale | none
	Duration  int    `yaml:"duration"`  // ms, default 200
	Easing    string `yaml:"easing"`    // ease | ease-in | ease-out | linear
	Direction string `yaml:"direction"` // left | right | up | down (для slide)
	Delay     int    `yaml:"delay"`     // ms
}

func (a *AnimateConfig) UnmarshalYAML(node *yaml.Node) error {
	// поддержка короткой формы: animate: fade
	if node.Kind == yaml.ScalarNode {
		var s string
		if err := node.Decode(&s); err != nil {
			return err
		}
		a.Type = s
		if a.Duration == 0 {
			a.Duration = 200
		}
		if a.Easing == "" {
			a.Easing = "ease"
		}
		return nil
	}
	type raw AnimateConfig
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*a = AnimateConfig(r)
	if a.Duration == 0 {
		a.Duration = 200
	}
	if a.Easing == "" {
		a.Easing = "ease"
	}
	if a.Type == "" {
		a.Type = "fade"
	}
	return nil
}

// ScreenNode — один узел дерева макета из screen yaml.
type ScreenNode struct {
	Component string            `yaml:"component"` // имя компонента (div, sl-button, layout...)
	Layout    string            `yaml:"layout"`    // row|column|grid
	Key       string            `yaml:"key"`
	Props     map[string]string `yaml:"props"`
	Bind      string            `yaml:"bind"` // привязка к состоянию
	Label     string            `yaml:"label"`
	Text      string            `yaml:"text"`
	On        map[string]string `yaml:"on"`
	Children  []*ScreenNode     `yaml:"children"`
	If        string            `yaml:"if"` // условие (путь состояния)
	Repeat    string            `yaml:"repeat"` // путь к списку для repeat
	Animate   *AnimateConfig    `yaml:"animate"` // анимация появления/исчезновения
}

// Screen — корень экрана.
type Screen struct {
	Screen  string         `yaml:"screen"`
	Name    string         `yaml:"name"`
	Root    *ScreenNode    `yaml:"root"`
	Layout  *ScreenNode    `yaml:"layout"`
	Animate *AnimateConfig `yaml:"animate"` // переход экрана
}

// DesignTokens — дизайн-токены темы
type DesignTokens struct {
	Colors     map[string]string `yaml:"colors"`
	Spacing    map[string]string `yaml:"spacing"`
	Radius     map[string]string `yaml:"radius"`
	Typography map[string]string `yaml:"typography"`
	Shadows    map[string]string `yaml:"shadows"`
	Animations map[string]string `yaml:"animations"`
}

func (d *DesignTokens) UnmarshalYAML(node *yaml.Node) error {
	// поддержка как вложенных (colors: {primary: "#fff"}), так и плоских (sl-color-primary-500: "#fff")
	type raw DesignTokens
	if err := node.Decode((*raw)(d)); err != nil {
		return err
	}
	// если все поля пустые, пробуем как плоский map
	if len(d.Colors) == 0 && len(d.Spacing) == 0 && len(d.Radius) == 0 && len(d.Typography) == 0 && len(d.Shadows) == 0 && len(d.Animations) == 0 {
		var flat map[string]string
		if err := node.Decode(&flat); err == nil && len(flat) > 0 {
			if d.Colors == nil {
				d.Colors = map[string]string{}
			}
			for k, v := range flat {
				d.Colors[k] = v
			}
		}
	}
	return nil
}

// Theme — конфиг темы (theme.yaml) с поддержкой дизайн-токенов и кастомных тем
type Theme struct {
	Active    string                       `yaml:"active"`
	Tokens    DesignTokens                 `yaml:"tokens"` // общие токены
	Themes    map[string]map[string]string `yaml:"themes"` // имя темы -> css-переменные (плоские, для обратной совместимости)
	RawThemes map[string]DesignTokens      `yaml:"-"`      // парсится вручную для вложенных
}

func (t *Theme) UnmarshalYAML(node *yaml.Node) error {
	type rawTheme Theme
	var r rawTheme
	if err := node.Decode(&r); err != nil {
		return err
	}
	*t = Theme(r)
	// парсим RawThemes отдельно для вложенных структур
	var raw map[string]yaml.Node
	if err := node.Decode(&raw); err == nil {
		if themesNode, ok := raw["themes"]; ok {
			var themesMap map[string]yaml.Node
			if err := themesNode.Decode(&themesMap); err == nil {
				t.RawThemes = map[string]DesignTokens{}
				for name, themeNode := range themesMap {
					var dt DesignTokens
					if err := themeNode.Decode(&dt); err == nil {
						// если dt пустой, но есть плоские ключи, это старый формат — пропускаем
						if len(dt.Colors) > 0 || len(dt.Spacing) > 0 || len(dt.Animations) > 0 {
							t.RawThemes[name] = dt
						}
					}
				}
			}
		}
	}
	if t.Active == "" {
		t.Active = "light"
	}
	return nil
}

// Loader загружает конфиги из директории проекта.
type Loader struct {
	Root string // корень проекта
}

// NewLoader создаёт загрузчик для корня проекта.
func NewLoader(root string) *Loader { return &Loader{Root: root} }

// LoadApp загружает и парсит app.yaml.
func (l *Loader) LoadApp() (*App, error) {
	var app App
	if err := readYAML(filepath.Join(l.Root, "app.yaml"), &app); err != nil {
		return nil, err
	}
	if app.Root == "" && app.ScreensDir == "" {
		app.ScreensDir = filepath.Join(l.Root, "screens")
	}
	if app.Root == "" {
		app.Root = "main"
	}
	if app.ThemePath == "" {
		app.ThemePath = filepath.Join(l.Root, "theme.yaml")
	}
	if app.ScreensDir == "" {
		app.ScreensDir = filepath.Join(l.Root, "screens")
	}
	if app.ThemeActive == "" {
		app.ThemeActive = "light"
	}
	return &app, nil
}

// LoadScreens загружает все экраны из папки экранов.
func (l *Loader) LoadScreens(dir string) (map[string]*Screen, error) {
	screens := map[string]*Screen{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		var sc Screen
		if err := readYAML(p, &sc); err != nil {
			return nil, fmt.Errorf("screen %s: %w", p, err)
		}
		if sc.Name == "" {
			sc.Name = strings.TrimSuffix(filepath.Base(p), ".yaml")
		}
		screens[sc.Name] = &sc
	}
	return screens, nil
}

// LoadTheme загружает theme.yaml.
func (l *Loader) LoadTheme(path string) (*Theme, error) {
	var t Theme
	if err := readYAML(path, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// LoadNet загружает net.yaml (если путь указан и файл существует).
func (l *Loader) LoadNet(path string) (*Net, error) {
	if path == "" {
		return nil, nil
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(l.Root, path)
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return nil, nil // net.yaml опционален
	}
	var n Net
	if err := readYAML(full, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// LoadState загружает state.yaml (если путь указан и файл существует).
func (l *Loader) LoadState(path string) (*State, error) {
	if path == "" {
		return nil, nil
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(l.Root, path)
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return nil, nil // state.yaml опционален
	}
	var s State
	if err := readYAML(full, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// LoadHooks загружает hooks.yaml (если путь указан и файл существует).
func (l *Loader) LoadHooks(path string) (*Hooks, error) {
	if path == "" {
		return nil, nil
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(l.Root, path)
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return nil, nil
	}
	var h Hooks
	if err := readYAML(full, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// LoadKeys загружает keys.yaml (если путь указан и файл существует).
func (l *Loader) LoadKeys(path string) (*Keys, error) {
	if path == "" {
		return nil, nil
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(l.Root, path)
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return nil, nil
	}
	var k Keys
	if err := readYAML(full, &k); err != nil {
		return nil, err
	}
	return &k, nil
}

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
