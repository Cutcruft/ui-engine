package config

// Keys — назначения сочетаний клавиш (keys.yaml).
type Keys struct {
	// Bindings — имя пары -> описание сочетания.
	Bindings map[string]KeyBinding `yaml:"bindings"`
}

// KeyBinding — одно сочетание клавиш.
type KeyBinding struct {
	// Combo — список сочетаний (например ["meta+s", "/"]).
	Combo  []string         `yaml:"combo"`
	Action string           `yaml:"action"` // ключ действия мини-DSL
	When   string           `yaml:"when"`   // условие (путь состояния)
	Args   []string         `yaml:"args"`   // аргументы действия
}
