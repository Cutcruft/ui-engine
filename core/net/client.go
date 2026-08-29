// Package net реализует декларативный клиент сокетного протокола
// из net.yaml: транспорт, события клиента/сервера, reconnect, маппинг
// результата в состояние.
//
// Логика клиента (client.go) не зависит от транспорта — обмен идёт через
// интерфейс Transport. Реализация WebSocket для браузера — в js_ws.go.
package net

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ui-engine/core/config"
)

// State — реактивное состояние (минимальный интерфейс для net).
type State interface {
	Set(path string, value any)
	GetString(path string) string
}

// Action — диспетчер действий (для on:-маппинга вида action.foo).
type Action interface {
	Dispatch(key string)
}

// Transport — низкоуровневый канал связи.
type Transport interface {
	// Connect открывает соединение. Колбэки для сообщения/открытия/закрытия.
	Connect(url string, onOpen func(), onMessage func([]byte), onClose func()) error
	// Send отправляет JSON-сообщение.
	Send(data []byte)
	// Close закрывает соединение.
	Close()
}

// Event — wire-сообщение.
type Event struct {
	Name    string          `json:"event"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client — декларативный сокетный клиент с надёжностью.
type Client struct {
	cfg    *config.Net
	state  State
	action Action
	tr     Transport

	mu      sync.Mutex
	closed  bool
	retries int

	// надёжность
	queue   []Event
	queueMu sync.Mutex
	pingMu  sync.Mutex
	lastPong time.Time
}

// New создаёт клиент из контракта.
func New(cfg *config.Net, st State, act Action, tr Transport) *Client {
	return &Client{cfg: cfg, state: st, action: act, tr: tr, closed: true}
}

// Connect открывает соединение (с учётом политики реконнекта).
func (c *Client) Connect() {
	c.mu.Lock()
	if !c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = false
	c.mu.Unlock()

	url := c.buildURL()
	err := c.tr.Connect(url,
		func() { c.onOpen() },
		func(data []byte) { c.onMessage(data) },
		func() { c.onClose() },
	)
	if err != nil {
		log.Printf("net: connect error: %v", err)
		c.scheduleReconnect()
	}
}

func (c *Client) buildURL() string {
	url := c.cfg.URL
	auth := c.cfg.Auth
	if auth == nil {
		return url
	}
	token := ""
	if auth.TokenRef != "" {
		token = c.state.GetString(auth.TokenRef)
	}
	if token == "" {
		return url
	}
	switch auth.Method {
	case "query":
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		key := auth.QueryKey
		if key == "" {
			key = "token"
		}
		return url + sep + key + "=" + token
	case "header":
		// WebSocket из браузера не позволяет задать заголовки — пропускаем.
		return url
	case "handshake":
		return url // токен отправляется событием auth после open
	default:
		return url
	}
}

func (c *Client) onOpen() {
	c.mu.Lock()
	c.retries = 0
	c.mu.Unlock()
	log.Println("net: connected")
	c.pingMu.Lock()
	c.lastPong = time.Now()
	c.pingMu.Unlock()

	// handshake: если auth-метод handshake — отправить токен событием.
	if a := c.cfg.Auth; a != nil && a.Method == "handshake" && a.TokenRef != "" {
		token := c.state.GetString(a.TokenRef)
		if token != "" {
			c.sendNow("auth", map[string]any{"token": token})
		}
	}

	// отправить очередь оффлайн-сообщений
	c.flushQueue()

	// ping keep-alive
	if c.cfg.Ping != nil {
		go c.startPing()
	}
	// подписка на изменение токена для auth refresh
	if a := c.cfg.Auth; a != nil && a.TokenRef != "" {
		// состояние уже подписано в wasm, но для надёжности — re-auth при изменении токена
		// (реализуется через отдельный watcher в wasm, здесь — заглушка)
	}
}

func (c *Client) startPing() {
	ping := c.cfg.Ping
	if ping == nil || ping.IntervalMS <= 0 {
		return
	}
	interval := time.Duration(ping.IntervalMS) * time.Millisecond
	timeout := time.Duration(ping.TimeoutMS) * time.Millisecond
	if timeout == 0 {
		timeout = interval * 2
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
		c.pingMu.Lock()
		sincePong := time.Since(c.lastPong)
		c.pingMu.Unlock()
		if sincePong > timeout {
			log.Println("net: ping timeout, reconnecting")
			c.tr.Close()
			return
		}
		c.sendNow("ping", nil)
	}
}

func (c *Client) flushQueue() {
	c.queueMu.Lock()
	q := c.queue
	c.queue = nil
	c.queueMu.Unlock()
	for _, ev := range q {
		wire, _ := json.Marshal(ev)
		c.tr.Send(wire)
	}
	if len(q) > 0 {
		log.Printf("net: flushed %d queued messages", len(q))
	}
}

// Send отправляет клиентское событие (с очередью при оффлайне).
func (c *Client) Send(name string, payload map[string]any) {
	ev := Event{Name: name}
	if payload != nil {
		data, _ := json.Marshal(payload)
		ev.Payload = data
	}
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		c.queueMu.Lock()
		c.queue = append(c.queue, ev)
		c.queueMu.Unlock()
		log.Printf("net: queued %s (offline, %d in queue)", name, len(c.queue))
		return
	}
	wire, err := json.Marshal(ev)
	if err != nil {
		log.Printf("net: marshal error: %v", err)
		return
	}
	c.tr.Send(wire)
}

// sendNow отправляет немедленно, без очереди (для ping/auth)
func (c *Client) sendNow(name string, payload map[string]any) {
	ev := Event{Name: name}
	if payload != nil {
		data, _ := json.Marshal(payload)
		ev.Payload = data
	}
	wire, _ := json.Marshal(ev)
	c.tr.Send(wire)
}

// SendEvent отправляет событие с заполнением payload из состояния (проекция схемы).
func (c *Client) SendEvent(name string) {
	ce, ok := c.cfg.Events.Client[name]
	if !ok {
		c.Send(name, nil)
		return
	}
	payload := map[string]any{}
	for k, f := range ce.Request {
		// читаем из состояния по пути state.<k> если существует
		if v, ok := c.lookupStateField(k); ok {
			payload[k] = v
		} else if f.Default != nil {
			payload[k] = f.Default
		} else if f.Required {
			payload[k] = zeroValue(f.Type)
		}
	}
	c.Send(name, payload)
}

// lookupStateField пробует прочитать value из state.path (поля запроса).
func (c *Client) lookupStateField(k string) (any, bool) {
	// Простое соглашение: ищем в состоянии по пути "state.<k>".
	v := c.state.GetString("state." + k)
	if v != "" {
		return v, true
	}
	return nil, false
}

func (c *Client) onMessage(data []byte) {
	c.pingMu.Lock()
	c.lastPong = time.Now()
	c.pingMu.Unlock()
	var ev Event
	if err := json.Unmarshal(data, &ev); err != nil {
		log.Printf("net: bad message: %v", err)
		return
	}
	// системные события
	switch ev.Name {
	case "pong":
		return
	case "auth_required", "auth_expired":
		// токен протух — пробуем refresh (если есть TokenRef) и переподключаемся
		if a := c.cfg.Auth; a != nil && a.TokenRef != "" {
			log.Println("net: auth required, refreshing")
			// в реальном продукте — вызов refresh endpoint, здесь — просто re-auth
			token := c.state.GetString(a.TokenRef)
			if token != "" {
				c.sendNow("auth", map[string]any{"token": token})
				return
			}
		}
		c.tr.Close()
		return
	case "ping":
		c.sendNow("pong", nil)
		return
	}
	c.handleServerEvent(ev)
}

func (c *Client) handleServerEvent(ev Event) {
	se, ok := c.cfg.Events.Server[ev.Name]
	if !ok {
		// неизвестное серверное событие — игнорируем (или обрабатываем специальные ping/pong)
		return
	}
	payload := map[string]any{}
	if len(ev.Payload) > 0 {
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			log.Printf("net: payload error: %v", err)
		}
	}
	for _, mapping := range se.On {
		c.applyMapping(mapping, payload)
	}
}

// applyMapping применяет инструкцию маппинга:
//
//	state.session.user = user        — скопировать payload.user в состояние
//	state.token = user.token         — скопировать вложенное поле
//	action.foo                       — вызвать действие
func (c *Client) applyMapping(m string, payload map[string]any) {
	m = strings.TrimSpace(m)
	if strings.HasPrefix(m, "action.") {
		if c.action != nil {
			c.action.Dispatch(m)
		}
		return
	}
	// парсим "path = source"
	eq := strings.Index(m, "=")
	if eq < 0 {
		return
	}
	path := strings.TrimSpace(m[:eq])
	src := strings.TrimSpace(m[eq+1:])
	val := resolvePayload(payload, src)
	c.state.Set(path, val)
}

// resolvePayload достаёт вложенное значение из payload по точечному пути.
func resolvePayload(payload map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var cur any = payload
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[p]
		if !ok {
			return nil
		}
	}
	return cur
}

func (c *Client) onClose() {
	c.mu.Lock()
	alreadyClosed := c.closed
	c.closed = true
	c.mu.Unlock()
	if alreadyClosed {
		return
	}
	log.Println("net: closed")
	c.scheduleReconnect()
}

func (c *Client) scheduleReconnect() {
	rc := c.cfg.Reconnect
	if rc != nil && rc.Attempts > 0 {
		c.mu.Lock()
		if c.retries >= rc.Attempts {
			c.mu.Unlock()
			log.Println("net: reconnect attempts exhausted")
			return
		}
		c.retries++
		c.mu.Unlock()
	}
	delay := c.reconnectDelay()
	log.Printf("net: reconnect in %v (attempt %d)", delay, c.retries+1)
	// неблокирующий реконнект с экспонентой
	go func() {
		time.Sleep(delay)
		c.Connect()
	}()
}

func (c *Client) reconnectDelay() time.Duration {
	rc := c.cfg.Reconnect
	base := 500
	max := 30000
	strategy := "exponential"
	if rc != nil {
		if rc.BaseMS > 0 {
			base = rc.BaseMS
		}
		if rc.MaxMS > 0 {
			max = rc.MaxMS
		}
		strategy = rc.Strategy
	}
	if strategy != "exponential" {
		return time.Duration(base) * time.Millisecond
	}
	c.mu.Lock()
	r := c.retries
	c.mu.Unlock()
	delay := base * (1 << uint(r))
	if delay > max || delay < 0 {
		delay = max
	}
	return time.Duration(delay) * time.Millisecond
}

// Close останавливает клиент.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	c.tr.Close()
}

func zeroValue(t string) any {
	switch t {
	case "int":
		return 0
	case "float":
		return 0.0
	case "bool":
		return false
	case "list":
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return ""
	}
}
