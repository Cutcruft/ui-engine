package config

// Net — контракт сокетного протокола (net.yaml).
type Net struct {
	URL       string         `yaml:"url"`       // wss://...
	Reconnect *Reconnect     `yaml:"reconnect"` // политика реконнекта
	Auth      *Auth          `yaml:"auth"`      // аутентификация
	Events    NetEvents      `yaml:"events"`    // события клиента и сервера
	Errors    map[string]NetError `yaml:"errors"`  // обработка ошибок
	Ping      *Ping          `yaml:"ping"`
}

// Reconnect — политика переподключения.
type Reconnect struct {
	Strategy string `yaml:"strategy"` // exponential | fixed
	BaseMS   int    `yaml:"base"`     // базовый интервал в мс
	MaxMS    int    `yaml:"max"`      // максимум в мс
	Attempts int    `yaml:"attempts"` // 0 = бесконечно
}

// Auth — конфигурация аутентификации.
type Auth struct {
	Method   string `yaml:"method"`   // handshake | query | header
	TokenRef string `yaml:"tokenRef"` // путь к токену в состоянии
	Header   string `yaml:"header"`   // имя заголовка (для header)
	QueryKey string `yaml:"queryKey"` // имя query-параметра (для query)
}

// NetEvents — события протокола.
type NetEvents struct {
	Client map[string]*ClientEvent `yaml:"client"` // события, отправляемые клиентом
	Server map[string]*ServerEvent `yaml:"server"` // события, принимаемые клиентом
}

// ClientEvent — событие, отправляемое клиентом (по контракту).
type ClientEvent struct {
	Request map[string]Field   `yaml:"request"` // схема запроса
	Reply   string             `yaml:"reply"`   // какое серверное событие ждём в ответ
	On      map[string]string  `yaml:"on"`      // действия маппинга результата -> состояние
	Emit    string             `yaml:"emit"`    // имя события, как отправляется (по умолчанию = имя)
}

// ServerEvent — событие, принимаемое клиентом.
type ServerEvent struct {
	Payload map[string]Field  `yaml:"payload"` // схема входящего payload
	On      map[string]string `yaml:"on"`      // маппинг payload -> состояние / действия
}

// Field — поле схемы (тип и опции).
type Field struct {
	Type     string         `yaml:"type"`     // string|int|float|bool|object|list
	Required bool           `yaml:"required"`
	Secret   bool           `yaml:"secret"`
	Default  any            `yaml:"default"`
	Items    *Field         `yaml:"items"`    // для list
	Fields   map[string]Field `yaml:"fields"` // для object
}

// NetError — описание ошибки и её обработка.
type NetError struct {
	Code   int              `yaml:"code"`
	Action map[string]string `yaml:"on"`
}

// Ping — настройка keep-alive.
type Ping struct {
	IntervalMS int    `yaml:"interval"` // интервал пинга в мс
	TimeoutMS  int    `yaml:"timeout"`  // таймаут ответа в мс
	OnTimeout  string `yaml:"onTimeout"` // действие при таймауте (reconnect и т.п.)
}

// EventNames возвращает имена серверных событий.
func (n *Net) ServerEventNames() []string {
	out := make([]string, 0, len(n.Events.Server))
	for k := range n.Events.Server {
		out = append(out, k)
	}
	return out
}

// ClientEventNames возвращает имена клиентских событий.
func (n *Net) ClientEventNames() []string {
	out := make([]string, 0, len(n.Events.Client))
	for k := range n.Events.Client {
		out = append(out, k)
	}
	return out
}
