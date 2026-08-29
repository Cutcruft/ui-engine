package config

// Hooks — веб-хуки приложения (hooks.yaml).
//
// Хуки — декларативные «куда что вызывать» на события жизненного цикла
// и внешние интеграции. Каждый хук выполняет список действий (мини-DSL),
// определённых в hooks.on.
type Hooks struct {
	// OnReady — действия при готовности приложения.
	OnReady []string `yaml:"onReady"`
	// OnMount — действия при монтировании.
	OnMount []string `yaml:"onMount"`
	// OnRoute — действия при смене экрана, key=имя экрана.
	OnRoute map[string][]string `yaml:"onRoute"`
	// External — внешние интеграции (HTTP/JS).
	External map[string]ExternalHook `yaml:"external"`
	// Actions — задекларированные действия (ключ -> список инструкций мини-DSL).
	Actions map[string][]string `yaml:"actions"`
}

// ExternalHook — внешний вызов (HTTP/JS-функция).
type ExternalHook struct {
	Endpoint string            `yaml:"endpoint"` // путь/URL
	Method   string            `yaml:"method"`   // GET|POST|...
	Headers  map[string]string `yaml:"headers"`
	Body     map[string]Field  `yaml:"body"` // схема тела
}
