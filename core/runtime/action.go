package runtime

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ui-engine/core/config"
)

// Sender — интерфейс отправки клиентских событий по сокету.
type Sender interface {
	SendEvent(name string)
}

// Handlers — сборка рантайма: сеть, хуки, клавиши, диспетчер.
type Handlers struct {
	engine  *Engine
	net     Sender
	keys    *config.Keys
	hooks   *config.Hooks
}

// Configure дополняет движок сетью, клавишами и хуками.
func (e *Engine) Configure(netCfg *config.Net, network Sender, keys *config.Keys, hooks *config.Hooks) {
	if network != nil {
		e.net = network
	}
	_ = netCfg
	e.keys = keys
	e.hooks = hooks
}

// Engine привязывает сеть (используется диспетчером).
func (e *Engine) BindNet(s Sender) { e.net = s }

// RunHooks выполняет список действий жизненного цикла.
func (e *Engine) RunHooks(actions []string) {
	for _, a := range actions {
		e.Dispatch(a)
	}
}

// OnRoute вызывается при смене экрана, выполняет хуки onRoute[name].
func (e *Engine) OnRoute(name string) {
	if e.hooks != nil {
		if acts := e.hooks.OnRoute[name]; len(acts) > 0 {
			e.RunHooks(acts)
		}
	}
}

// Keys возвращает текущую конфигурацию клавиш.
func (e *Engine) Keys() *config.Keys { return e.keys }

// executeAction выполняет одну инструкцию мини-DSL:
//
//	set <path> = <value>            — установить значение состояния
//	toggle <path>                   — инвертировать bool
//	inc <path>                      — прибавить 1
//	action.<key> / action <key>     — вызвать зарегистрированное действие
//	navigate <screen> / screen <s>  — переключить экран
//	net.send <event>                — отправить клиентское событие по сокету
//	net.connect                     — подключить сокет (принудительно)
func (e *Engine) executeAction(dsl string) {
	dsl = strings.TrimSpace(dsl)
	if dsl == "" {
		return
	}
	// action.<key> — префиксный вызов (без пробела)
	if strings.HasPrefix(dsl, "action.") {
		key := strings.TrimSpace(strings.TrimPrefix(dsl, "action."))
		// отрезаем возможные аргументы после ключа
		if sp := strings.Index(key, " "); sp >= 0 {
			key = strings.TrimSpace(key[:sp])
		}
		e.DispatchKey(key)
		return
	}
	fields := strings.Fields(dsl)
	if len(fields) == 0 {
		return
	}

	verb := fields[0]
	switch verb {
	case "set":
		if len(fields) >= 3 && fields[1] == "=" {
			// set = value (без пробелов вокруг =) => fields[1] == "x"?
		}
		e.execSet(dsl)
	case "toggle":
		if len(fields) >= 2 {
			e.toggle(fields[1])
		}
	case "inc":
		if len(fields) >= 2 {
			e.inc(fields[1])
		}
	case "push":
		e.execPush(dsl)
	case "remove":
		if len(fields) >= 2 {
			e.execRemove(fields[1])
			// поддержка remove <path> <index>  (например remove state.todos 1)
			if len(fields) >= 3 {
				// если второй аргумент — индекс, а первый — путь к списку
				// пробуем как remove state.todos 1
				if idx, err := strconv.Atoi(fields[2]); err == nil {
					e.removeAt(fields[1], idx)
					return
				}
			}
			// иначе remove <path> где path уже содержит индекс: remove state.todos.1
		}
	case "navigate", "screen":
		if len(fields) >= 2 {
			e.SetScreen(fields[1])
			e.OnRoute(fields[1])
		}
	case "action":
		if len(fields) >= 2 {
			e.DispatchKey(fields[1])
		}
	case "net":
		if len(fields) >= 2 {
			e.execNet(fields[1], fields[2:])
		}
	}
}

// execSet парсит "set <path> = <value>" и устанавливает значение.
func (e *Engine) execSet(dsl string) {
	rest := strings.TrimSpace(strings.TrimPrefix(dsl, "set"))
	eq := strings.Index(rest, "=")
	if eq < 0 {
		return
	}
	path := strings.TrimSpace(rest[:eq])
	val := strings.TrimSpace(rest[eq+1:])
	e.state.Set(path, parseValue(val))
}

func (e *Engine) execPush(dsl string) {
	rest := strings.TrimSpace(strings.TrimPrefix(dsl, "push"))
	if rest == "" {
		return
	}
	// push <path> <json>  — поддерживаем {{state.xxx}} внутри JSON
	sp := strings.Index(rest, " ")
	if sp < 0 {
		return
	}
	path := strings.TrimSpace(rest[:sp])
	valStr := strings.TrimSpace(rest[sp+1:])
	// подставляем шаблоны {{state.xxx}} / {{item.xxx}} / {{index}} если есть
	if strings.Contains(valStr, "{{") {
		valStr = e.resolveText(valStr)
	}
	val := parseValue(valStr)
	if s, ok := val.(string); ok && len(s) > 0 && (s[0] == '{' || s[0] == '[') {
		var jv any
		if err := json.Unmarshal([]byte(s), &jv); err == nil {
			val = jv
		}
	}
	cur, _ := e.state.Get(path)
	var arr []any
	if cur != nil {
		// пробуем привести к слайсу через type switch
		switch t := cur.(type) {
		case []any:
			arr = t
		case []map[string]any:
			for _, m := range t {
				arr = append(arr, m)
			}
		default:
			// если не слайс, делаем пустой
			arr = []any{}
		}
	}
	arr = append(arr, val)
	e.state.Set(path, arr)
}

func (e *Engine) execRemove(path string) {
	// path вида state.todos.1  -> удалить элемент по индексу
	// разбиваем по точкам, последний сегмент — индекс
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return
	}
	last := parts[len(parts)-1]
	idx, err := strconv.Atoi(last)
	if err != nil {
		return
	}
	basePath := strings.Join(parts[:len(parts)-1], ".")
	cur, ok := e.state.Get(basePath)
	if !ok {
		return
	}
	var arr []any
	switch t := cur.(type) {
	case []any:
		arr = t
	default:
		return
	}
	if idx < 0 || idx >= len(arr) {
		return
	}
	arr = append(arr[:idx], arr[idx+1:]...)
	e.state.Set(basePath, arr)
}

func (e *Engine) removeAt(path string, idx int) {
	cur, ok := e.state.Get(path)
	if !ok {
		return
	}
	var arr []any
	switch t := cur.(type) {
	case []any:
		arr = t
	default:
		return
	}
	if idx < 0 || idx >= len(arr) {
		return
	}
	arr = append(arr[:idx], arr[idx+1:]...)
	e.state.Set(path, arr)
}

func (e *Engine) toggle(path string) {
	if v, ok := e.state.Get(path); ok {
		switch t := v.(type) {
		case bool:
			e.state.Set(path, !t)
		case string:
			if t == "true" {
				e.state.Set(path, "false")
			} else {
				e.state.Set(path, "true")
			}
		}
	} else if e.state.GetString(path) == "" {
		e.state.Set(path, true)
	}
}

func (e *Engine) inc(path string) {
	if v, ok := e.state.Get(path); ok {
		switch t := v.(type) {
		case int:
			e.state.Set(path, t+1)
		case float64:
			e.state.Set(path, t+1)
		case string:
			if n, err := strconv.Atoi(t); err == nil {
				e.state.Set(path, n+1)
			}
		}
	} else {
		e.state.Set(path, 1)
	}
}

func (e *Engine) execNet(sub string, args []string) {
	switch sub {
	case "send":
		if len(args) >= 1 {
			if e.net != nil {
				e.net.SendEvent(args[0])
			}
		}
	case "connect":
		if c, ok := e.net.(interface{ Connect() }); ok {
			c.Connect()
		}
	}
}

// parseValue приводит строку к нужному типу (число/булево/строка/JSON).
func parseValue(s string) any {
	s = strings.TrimSpace(s)
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && strings.Contains(s, ".") {
		return f
	}
	// JSON-объект/массив для списков
	if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
		var jv any
		if err := json.Unmarshal([]byte(s), &jv); err == nil {
			return jv
		}
	}
	return s
}
