// Package state — декларативная модель состояния приложения.
//
// Состояние хранится как вложенный map[string]any, доступ по путям
// ("state.form.login.email"). Компоненты подписываются на префиксы путей;
// при изменении состояния подписчики уведомляются и runtime пере-рендерит
// затронутые поддеревья.
package state

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Store — реактивное хранилище состояния.
type Store struct {
	mu      sync.RWMutex
	data    map[string]any
	subs    map[string]map[int]func()
	nextSub int
}

// New создаёт пустое состояние.
func New() *Store {
	return &Store{
		data: map[string]any{},
		subs: map[string]map[int]func(){},
	}
}

// Set записывает значение по пути ("state.form.login.email") и уведомляет
// всех подписчиков затронутых префиксов. path может содержать сегменты.
func (s *Store) Set(path string, value any) {
	s.mu.Lock()
	segments := strings.Split(path, ".")
	if len(segments) > 0 && segments[0] == "state" {
		segments = segments[1:]
	}
	s.setPath(s.data, segments, value)
	// собрать подписчиков
	notify := s.collectSubscribers("state." + strings.Join(segments, "."))
	s.mu.Unlock()
	for _, fn := range notify {
		fn()
	}
}

func (s *Store) setPath(m map[string]any, path []string, value any) {
	if len(path) == 1 {
		m[path[0]] = value
		return
	}
	// если следующий сегмент — индекс массива, а текущее значение — слайс
	if idx, err := parseIndex(path[1]); err == nil {
		if arr, ok := m[path[0]].([]any); ok {
			// путь вида todos.0.xxx  -> arr[0] это map
			if idx < 0 || idx >= len(arr) {
				return
			}
			if len(path) == 2 {
				// todos.0 = value (замена элемента)
				arr[idx] = value
				m[path[0]] = arr
				return
			}
			// todos.0.xxx -> рекурсия в элемент
			if elem, ok := arr[idx].(map[string]any); ok {
				s.setPath(elem, path[2:], value)
				return
			}
			return
		}
	}
	next, ok := m[path[0]].(map[string]any)
	if !ok {
		next = map[string]any{}
		m[path[0]] = next
	}
	s.setPath(next, path[1:], value)
}

func parseIndex(s string) (int, error) {
	// простой парсер без импорта strconv для избежания цикла, но используем fmt
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not index")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// Get возвращает значение по пути.
func (s *Store) Get(path string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	segments := strings.Split(path, ".")
	if segments[0] == "state" {
		segments = segments[1:]
	}
	return s.getPath(s.data, segments)
}

func (s *Store) getPath(m map[string]any, path []string) (any, bool) {
	if len(path) == 0 {
		return m, true
	}
	cur, ok := m[path[0]]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return cur, true
	}
	// поддержка индекса массива: todos.0.title
	if idx, err := parseIndex(path[1]); err == nil {
		if arr, ok := cur.([]any); ok {
			if idx < 0 || idx >= len(arr) {
				return nil, false
			}
			if len(path) == 2 {
				return arr[idx], true
			}
			if elem, ok := arr[idx].(map[string]any); ok {
				return s.getPath(elem, path[2:])
			}
			return nil, false
		}
	}
	next, ok := cur.(map[string]any)
	if !ok {
		return nil, false
	}
	return s.getPath(next, path[1:])
}

// GetString удобное чтение строки.
func (s *Store) GetString(path string) string {
	v, ok := s.Get(path)
	if !ok {
		return ""
	}
	if sv, ok := v.(string); ok {
		return sv
	}
	return fmt.Sprintf("%v", v)
}

// Subscribe подписывается на префикс пути. fn вызывается при любом изменении
// на этом пути или внутри него. Возвращает id для Unsubscribe.
func (s *Store) Subscribe(prefix string, fn func()) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSub++
	if _, ok := s.subs[prefix]; !ok {
		s.subs[prefix] = map[int]func(){}
	}
	s.subs[prefix][s.nextSub] = fn
	return s.nextSub
}

// Unsubscribe отписывает подписчика.
func (s *Store) Unsubscribe(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.subs {
		delete(m, id)
	}
}

func (s *Store) collectSubscribers(changed string) []func() {
	var notify []func()
	// Подписчики на сам путь и на все его префиксы.
	parts := strings.Split(changed, ".")
	for i := len(parts); i >= 1; i-- {
		prefix := strings.Join(parts[:i], ".")
		if m, ok := s.subs[prefix]; ok {
			// детерминированный порядок
			ids := make([]int, 0, len(m))
			for id := range m {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			for _, id := range ids {
				notify = append(notify, m[id])
			}
		}
	}
	// Также подписчики на "state" (корень), если префикс не равен ему.
	if changed != "state" {
		if m, ok := s.subs["state"]; ok {
			ids := make([]int, 0, len(m))
			for id := range m {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			for _, id := range ids {
				notify = append(notify, m[id])
			}
		}
	}
	return notify
}

// Snapshot возвращает глубокую копию всего состояния.
func (s *Store) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMap(s.data)
}

func copyMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if cm, ok := v.(map[string]any); ok {
			out[k] = copyMap(cm)
		} else {
			out[k] = v
		}
	}
	return out
}
