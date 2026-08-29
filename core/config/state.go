package config

// State — декларативная модель состояния (state.yaml).
type State struct {
	// Vars — переменные состояния: путь/имя -> описание переменной.
	Vars map[string]Variable `yaml:"state"`
}

// Variable — описание одной переменной состояния.
type Variable struct {
	Type     string            `yaml:"type"`   // string|int|float|bool|object|list
	Required bool              `yaml:"required"`
	Secret   bool              `yaml:"secret"` // не рендерить (пароли/токены)
	Default  any               `yaml:"default"`
	Items    *Variable         `yaml:"items"`  // для list: тип элементов
	Fields   map[string]Variable `yaml:"fields"` // для object: поля
}