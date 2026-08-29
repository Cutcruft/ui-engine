// Package runtime связывает конфиг, состояние, VDOM и DOM-рендерер.
// Здесь: event bus, выполнение действий (мини-DSL) и построение VNode
// из декларативного макета (ScreenNode).
package runtime

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ui-engine/core/config"
	"github.com/ui-engine/core/diff"
	"github.com/ui-engine/core/vdom"
)

// Renderer — интерфейс применения патчей (реализует dom.Renderer).
type Renderer interface {
	Mount(root *vdom.VNode)
	Patch(ops []diff.Op)
}

// Screens — интерфейс доступа к важным экранам.
type Screens interface {
	Get(name string) (*config.Screen, bool)
}

// Engine — главный компонент рантайма приложения.
type Engine struct {
	cfg    *config.App
	screens map[string]*config.Screen
	state  State
	// actions — реестр действий (actionKey -> обработчик)
	actions map[string]func()
	// root — текущее VNode-дерево для диффа
	rootID    string
	prevRoot  *vdom.VNode
	renderer  Renderer
	// net — сокетный клиент (опционально)
	net Sender
	// keys — конфигурация горячих клавиш
	keys *config.Keys
	// hooks — веб-хуки
	hooks *config.Hooks
	// reactive — узлы, зависящие от состояния (bind/{{...}}/if). При изменении
	// затронутого префикса такие узлы пере-рендерятся точечно (reactivity
	// по поддеревьям), без полного diff всего дерева.
	reactive []*reactiveNode
	// rendering — защита от повторного входа (подписки не эмитят во время сборки).
	rendering bool
}

// reactiveNode описывает один узел макета, зависящий от состояния.
type reactiveNode struct {
	// screen — исходный узел макета (для пере-рендера).
	screen *config.ScreenNode
	// path — путь к узлу в DOM-дереве (индексы детей от корня).
	path []int
	// prev — предыдущий VNode узла (для частичного diff).
	prev *vdom.VNode
	// prefixes — префиксы состояния (bind/if/{{...}}), на которые подписаны.
	prefixes []string
	// subIDs — идентификаторы подписок (для снятия при пере-рендере).
	subIDs []int
	// dirty — узел требует пере-рендера (сработала подписка).
	dirty bool
}

// State — интерфейс состояния, используемый runtime.
type State interface {
	vdom.StateReader
	Set(path string, value any)
	GetString(path string) string
	Subscribe(prefix string, fn func()) int
	Unsubscribe(id int)
}

// NewEngine создаёт движок.
func NewEngine(cfg *config.App, screens map[string]*config.Screen, st State) *Engine {
	return &Engine{
		cfg:      cfg,
		screens:  screens,
		state:    st,
		actions:  map[string]func(){},
		reactive: []*reactiveNode{},
	}
}

// RegisterAction регистрирует действие (из YAML on: или хуков).
func (e *Engine) RegisterAction(key string, fn func()) {
	e.actions[key] = fn
}

// Dispatch выполняет инструкцию мини-DSL (например строки из on: или хуков).
func (e *Engine) Dispatch(key string) {
	e.executeAction(key)
}

// DispatchKey вызывает зарегистрированное действие по точному ключу (action.<...>).
func (e *Engine) DispatchKey(key string) {
	if fn, ok := e.actions[key]; ok {
		fn()
		return
	}
	// fallback: без префикса action.
	if fn, ok := e.actions["action."+key]; ok {
		fn()
	}
}

// SetScreen переключает активный экран.
func (e *Engine) SetScreen(name string) {
	e.cfg.Root = name
	e.Apply()
}

// SetRenderer привязывает DOM-рендерер (после монтирования корня).
func (e *Engine) SetRenderer(r Renderer) { e.renderer = r }

// Mount монтирует текущее дерево целиком (без диффа) и подписывает
// реактивные узлы на состояние.
func (e *Engine) Mount() {
	e.rootID = e.cfg.Root
	e.rebuildRoot()
	if e.renderer != nil {
		e.renderer.Mount(e.prevRoot)
	}
}

// Apply применяет изменения. Если есть "грязные" реактивные узлы — точечно
// пере-рендерит только их поддеревья (реактивность по подпискам); иначе —
// полный diff (навигация и прочие не-реактивные изменения).
func (e *Engine) Apply() {
	if e.renderer == nil {
		return
	}
	if e.applyReactive() {
		// Точечный пере-рендер уже пропатчил DOM; дерево в памяти перестраиваем
		// целиком для согласованности (DOM не трогаем).
		e.rebuildRoot()
		return
	}
	e.fullApply()
}

// fullApply применяет полный diff и заново подписывает реактивные узлы.
func (e *Engine) fullApply() {
	e.unsubscribeAll()
	e.reactive = e.reactive[:0]
	next := e.Render()
	ops := diff.Diff(e.prevRoot, next, nil)
	if len(ops) > 0 {
		e.renderer.Patch(ops)
	}
	if e.rootID == e.cfg.Root {
		e.prevRoot = next
	} else {
		e.rootID = e.cfg.Root
		e.prevRoot = next
		e.renderer.Mount(next)
	}
	for _, rn := range e.reactive {
		rn.prev = e.nodeAtPath(next, rn.path)
	}
}

// applyReactive пере-рендерит "грязные" реактивные поддеревья частичным diff.
// Возвращает true, если хотя бы один узел был пере-рендерен точечно.
func (e *Engine) applyReactive() bool {
	applied := false
	for _, rn := range e.reactive {
		if !rn.dirty {
			continue
		}
		rn.dirty = false
		e.rendering = true
		next := e.buildNode(rn.screen, cloneInts(rn.path))
		e.rendering = false
		if rn.prev == nil {
			rn.prev = next
			continue
		}
		ops := diff.Diff(rn.prev, next, cloneInts(rn.path))
		if len(ops) > 0 {
			e.renderer.Patch(ops)
		}
		rn.prev = next
		applied = true
	}
	return applied
}

// rebuildRoot пересобирает дерево целиком, снимает старые подписки реактивных
// узлов и подписывает заново. Возвращает true, если пере-рендер произошёл
// через регенерацию (используется в Apply для избежания двойного diff).
func (e *Engine) rebuildRoot() bool {
	e.unsubscribeAll()
	e.reactive = e.reactive[:0]
	next := e.Render()
	e.prevRoot = next
	// после сборки реактивные узлы получили prev=nil; актуализируем их из дерева
	for _, rn := range e.reactive {
		rn.prev = e.nodeAtPath(next, rn.path)
	}
	return false
}

// unsubscribeAll снимает все подписки реактивных узлов.
func (e *Engine) unsubscribeAll() {
	for _, rn := range e.reactive {
		for _, id := range rn.subIDs {
			e.state.Unsubscribe(id)
		}
	}
}

// Render строит VNode текущего экрана.
func (e *Engine) Render() *vdom.VNode {
	sc, ok := e.screens[e.cfg.Root]
	if !ok {
		return vdom.NewElement("div", "error").WithText("screen not found: " + e.cfg.Root)
	}
	var root *config.ScreenNode
	if sc.Layout != nil {
		root = sc.Layout
	} else {
		root = sc.Root
	}
	if root == nil {
		return vdom.NewElement("div", "empty").WithText("empty screen")
	}
	node := e.buildNode(root, []int{})
	// screen-level animate (transition)
	if sc.Animate != nil && node.Animate == nil {
		node.Animate = &vdom.Animate{
			Type:      sc.Animate.Type,
			Duration:  sc.Animate.Duration,
			Easing:    sc.Animate.Easing,
			Direction: sc.Animate.Direction,
			Delay:     sc.Animate.Delay,
		}
		node.WithProp("data-animate", sc.Animate.Type)
		node.WithProp("data-animate-duration", fmt.Sprintf("%d", sc.Animate.Duration))
	}
	return node
}

// buildNode рекурсивно строит VNode из ScreenNode макета.
func (e *Engine) buildNode(n *config.ScreenNode, path []int) *vdom.VNode {
	comp := n.Component
	if comp == "" && n.Layout != "" {
		comp = n.Layout // row/column/grid — делегируется layout-плагину
	}
	if comp == "" {
		comp = "div"
	}

	key := n.Key
	if key == "" {
		key = strings.Join(pathInts(path), "_")
	}

	// условие if: если указан путь и он пуст/false — рендерим скрытый placeholder,
	// чтобы путь оставался стабильным и реактивность работала (hidden -> visible).
	if n.If != "" {
		if v, ok := e.state.Get(n.If); ok {
			if isFalsy(v) {
				ph := vdom.NewElement(comp, key)
				ph.WithProp("style", "display:none")
				ph.WithProp("data-if", "hidden")
				if !e.rendering {
					e.registerReactive(n, path)
				}
				return ph
			}
		}
	}

	// repeat: список — контейнер, чьи дети-шаблоны повторяются per item
	if n.Repeat != "" {
		// для repeat контейнер — всегда generic, без привязки к конкретному типу
		node := vdom.NewElement(comp, key)
		for k, v := range n.Props {
			node.WithProp(k, v)
		}
		// текст контейнера с item (редко, но поддержим)
		if n.Text != "" {
			// repeat-контейнер сам не использует item, только его дети
			node.WithText(e.resolveText(n.Text))
		} else if n.Label != "" {
			node.WithText(e.resolveText(n.Label))
		}
		if n.Bind != "" && n.Text == "" && n.Label == "" {
			if comp == "sl-input" || comp == "input" {
				node.WithProp("value", e.state.GetString(n.Bind))
			} else {
				node.WithText(e.state.GetString(n.Bind))
			}
		}
		for ev, act := range n.On {
			node.WithEvent(ev, act)
		}
		// развернуть список
		if listVal, ok := e.state.Get(n.Repeat); ok {
			if arr := toSlice(listVal); arr != nil {
				for idx, item := range arr {
					for ti, tmpl := range n.Children {
						// путь для этого экземпляра: контейнер-путь + индекс в развернутом списке
						childIdx := idx*len(n.Children) + ti
						childPath := append(cloneInts(path), childIdx)
						// ключ делаем уникальным per item
						tmplKey := tmpl.Key
						if tmplKey == "" {
							tmplKey = fmt.Sprintf("%d", ti)
						}
						// попытаться взять id из item для стабильного ключа
						if id := getItemField(item, "id"); id != nil {
							tmplKey = fmt.Sprintf("%s_%v", tmpl.Key, id)
						} else {
							tmplKey = fmt.Sprintf("%s_%d", tmpl.Key, idx)
						}
						// клонируем шаблон с уникальным ключом для этого item
						cloned := cloneScreenNode(tmpl)
						cloned.Key = tmplKey
						child := e.buildNodeWithItem(cloned, childPath, item, idx)
						if child != nil {
							node.WithChild(child)
						}
					}
				}
			}
		}
		if !e.rendering {
			e.registerReactive(n, path)
		}
		return node
	}

	// Строим элемент по типу компонента — строго generic, без хардкода.
	// Нативные HTML-теги рендерятся как есть, остальные — делегируются плагинам через JS (window.UIEngine.getComponent).
	// richtext — особый случай: рендерим как div с contenteditable (fallback, если плагин не загрузился — плагин переопределит)
	var node *vdom.VNode
	if comp == "richtext" {
		node = vdom.NewElement("div", key)
		node.WithProp("contenteditable", "true")
		node.WithProp("class", "ui-richtext")
		if ph, ok := n.Props["placeholder"]; ok {
			node.WithProp("data-placeholder", ph)
		}
	} else {
		native := map[string]bool{"div": true, "span": true, "input": true, "button": true, "textarea": true, "select": true, "form": true, "label": true, "p": true, "h1": true, "h2": true, "h3": true, "ul": true, "li": true, "a": true, "img": true, "sl-button": true, "sl-card": true, "sl-alert": true, "sl-input": true}
		if native[comp] {
			node = vdom.NewElement(comp, key)
		} else {
			node = vdom.NewElement(comp, key)
		}
	}

	// props
	for k, v := range n.Props {
		node.WithProp(k, v)
	}

	// текст-контент (label/text)
	if n.Text != "" {
		if comp == "richtext" {
			node.WithProp("innerHTML", e.resolveText(n.Text))
		} else {
			node.WithText(e.resolveText(n.Text))
		}
	} else if n.Label != "" {
		if comp == "richtext" {
			node.WithProp("innerHTML", e.resolveText(n.Label))
		} else {
			node.WithText(e.resolveText(n.Label))
		}
	}

	// bind: если есть bind и нет текста — подставляем значение состояния как текст, value для input или innerHTML для richtext.
	if n.Bind != "" && n.Text == "" && n.Label == "" {
		if comp == "sl-input" || comp == "input" {
			node.WithProp("value", e.state.GetString(n.Bind))
		} else if comp == "richtext" {
			node.WithProp("innerHTML", e.state.GetString(n.Bind))
		} else {
			node.WithText(e.state.GetString(n.Bind))
		}
	}

	// события
	for ev, act := range n.On {
		node.WithEvent(ev, act)
	}

	// children
	for i, c := range n.Children {
		child := e.buildNode(c, append(path, i))
		if child != nil {
			node.WithChild(child)
		}
	}

	// анимация
	if n.Animate != nil {
		node.Animate = &vdom.Animate{
			Type:      n.Animate.Type,
			Duration:  n.Animate.Duration,
			Easing:    n.Animate.Easing,
			Direction: n.Animate.Direction,
			Delay:     n.Animate.Delay,
		}
		// также прокидываем как data-атрибуты для CSS
		node.WithProp("data-animate", n.Animate.Type)
		node.WithProp("data-animate-duration", fmt.Sprintf("%d", n.Animate.Duration))
		if n.Animate.Easing != "" {
			node.WithProp("data-animate-easing", n.Animate.Easing)
		}
		if n.Animate.Direction != "" {
			node.WithProp("data-animate-direction", n.Animate.Direction)
		}
	}

	// Реактивность: если узел зависит от состояния (bind / if / {{...}}),
	// регистрируем подписку на его префиксы — при изменении состояние
	// триггерит точечный пере-рендер поддерева.
	if !e.rendering {
		e.registerReactive(n, path)
	}

	return node
}

// nodePrefixes возвращает префиксы состояния, от которых зависит узел:
// bind-путь, if-путь, repeat-путь и все {{state.x}} внутри текстов/подписей.
func (e *Engine) nodePrefixes(n *config.ScreenNode) []string {
	var out []string
	if n.Bind != "" {
		out = append(out, n.Bind)
	}
	if n.If != "" {
		out = append(out, n.If)
	}
	if n.Repeat != "" {
		out = append(out, n.Repeat)
	}
	text := n.Text
	if text == "" {
		text = n.Label
	}
	for _, p := range e.templatePaths(text) {
		out = append(out, p)
	}
	return out
}

// templatePaths извлекает пути state.* внутри {{ }} из строки.
func (e *Engine) templatePaths(s string) []string {
	var out []string
	if !strings.Contains(s, "{{") {
		return out
	}
	rest := s
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			break
		}
		rest = rest[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			break
		}
		path := strings.TrimSpace(rest[:end])
		if path != "" {
			out = append(out, path)
		}
		rest = rest[end+2:]
	}
	return out
}

// isReactiveNode определяет, зависит ли узел от состояния.
func (e *Engine) isReactiveNode(n *config.ScreenNode) bool {
	if n.Bind != "" || n.If != "" || n.Repeat != "" {
		return true
	}
	if strings.Contains(n.Text, "{{") || strings.Contains(n.Label, "{{") {
		return true
	}
	return false
}

// registerReactive подписывает узел n (по пути path) на его префиксы состояния.
func (e *Engine) registerReactive(n *config.ScreenNode, path []int) {
	if !e.isReactiveNode(n) {
		return
	}
	rn := &reactiveNode{
		screen:   n,
		path:     cloneInts(path),
		prefixes: e.nodePrefixes(n),
	}
	for _, p := range rn.prefixes {
		if p == "" {
			continue
		}
		id := e.state.Subscribe(p, func() {
			rn.dirty = true
		})
		rn.subIDs = append(rn.subIDs, id)
	}
	e.reactive = append(e.reactive, rn)
}

// nodeAtPath возвращает VNode по пути индексов детей внутри дерева.
func (e *Engine) nodeAtPath(root *vdom.VNode, path []int) *vdom.VNode {
	cur := root
	for _, i := range path {
		if cur == nil || i < 0 || i >= len(cur.Children) {
			return nil
		}
		cur = cur.Children[i]
	}
	return cur
}

func cloneInts(p []int) []int {
	if p == nil {
		return []int{}
	}
	c := make([]int, len(p))
	copy(c, p)
	return c
}

// resolveText подставляет значения состояния в тексте вида {{state.x}}.
func (e *Engine) resolveText(t string) string {
	if !strings.Contains(t, "{{") {
		return t
	}
	var b strings.Builder
	rest := t
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:start])
		rest = rest[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			b.WriteString(rest)
			break
		}
		path := strings.TrimSpace(rest[:end])
		b.WriteString(e.state.GetString(path))
		rest = rest[end+2:]
	}
	return b.String()
}

func isFalsy(v any) bool {
	switch t := v.(type) {
	case string:
		return t == "" || t == "false" || t == "0"
	case bool:
		return !t
	case nil:
		return true
	case int:
		return t == 0
	case int64:
		return t == 0
	case float64:
		return t == 0
	case float32:
		return t == 0
	}
	return false
}

// toSlice приводит значение списка из state к []any (поддерживает []any, []map[string]any и т.п. через reflection).
func toSlice(v any) []any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		n := rv.Len()
		out := make([]any, n)
		for i := 0; i < n; i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	return nil
}

// getItemField достаёт поле item по пути "item.xxx" или "index".
func getItemField(item any, field string) any {
	if field == "index" {
		return nil
	}
	// field уже без префикса "item.", сюда приходит "id", "title" и т.д.
	if m, ok := item.(map[string]any); ok {
		if v, ok := m[field]; ok {
			return v
		}
		return nil
	}
	rv := reflect.ValueOf(item)
	if rv.Kind() == reflect.Map {
		// пробуем string ключ
		mv := rv.MapIndex(reflect.ValueOf(field))
		if mv.IsValid() {
			return mv.Interface()
		}
	}
	if rv.Kind() == reflect.Struct {
		fv := rv.FieldByName(field)
		if fv.IsValid() {
			return fv.Interface()
		}
		// пробуем Title-case
		fv = rv.FieldByName(strings.Title(field))
		if fv.IsValid() {
			return fv.Interface()
		}
	}
	return nil
}

// cloneScreenNode делает поверхностную копию узла (для уникализации ключа per item).
func cloneScreenNode(n *config.ScreenNode) *config.ScreenNode {
	if n == nil {
		return nil
	}
	c := *n
	if n.Props != nil {
		c.Props = make(map[string]string, len(n.Props))
		for k, v := range n.Props {
			c.Props[k] = v
		}
	}
	if n.On != nil {
		c.On = make(map[string]string, len(n.On))
		for k, v := range n.On {
			c.On[k] = v
		}
	}
	// Children не клонируем глубоко — они будут обработаны отдельно с item
	return &c
}

// resolveTextWithItem подставляет {{item.xxx}}, {{index}} и {{state.xxx}}.
func (e *Engine) resolveTextWithItem(t string, item any, index int) string {
	if !strings.Contains(t, "{{") {
		return t
	}
	var b strings.Builder
	rest := t
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:start])
		rest = rest[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			b.WriteString(rest)
			break
		}
		path := strings.TrimSpace(rest[:end])
		var val string
		switch {
		case path == "index":
			val = fmt.Sprintf("%d", index)
		case strings.HasPrefix(path, "item."):
			field := strings.TrimPrefix(path, "item.")
			if v := getItemField(item, field); v != nil {
				val = fmt.Sprintf("%v", v)
			}
		case path == "item":
			val = fmt.Sprintf("%v", item)
		default:
			val = e.state.GetString(path)
		}
		b.WriteString(val)
		rest = rest[end+2:]
	}
	return b.String()
}

// buildNodeWithItem строит VNode из шаблона с контекстом конкретного элемента списка.
func (e *Engine) buildNodeWithItem(n *config.ScreenNode, path []int, item any, index int) *vdom.VNode {
	// if с поддержкой item
	if n.If != "" {
		p := n.If
		var v any
		var ok bool
		if p == "item" {
			v = item
			ok = true
		} else if strings.HasPrefix(p, "item.") {
			field := strings.TrimPrefix(p, "item.")
			v = getItemField(item, field)
			ok = v != nil
			// если поле не найдено, считаем falsy
			if !ok {
				v = nil
			} else {
				ok = true
			}
		} else if p == "index" {
			v = index
			ok = true
		} else {
			v, ok = e.state.Get(p)
		}
		if ok && isFalsy(v) {
			ph := vdom.NewElement(n.Component, n.Key)
			if ph.Type == "" {
				ph.Type = "div"
			}
			ph.WithProp("style", "display:none")
			ph.WithProp("data-if", "hidden")
			return ph
		}
		if !ok && p != "" {
			// путь не найден — считаем falsy для item-полей
			if strings.HasPrefix(p, "item.") {
				ph := vdom.NewElement(n.Component, n.Key)
				if ph.Type == "" {
					ph.Type = "div"
				}
				ph.WithProp("style", "display:none")
				return ph
			}
		}
	}
	comp := n.Component
	if comp == "" && n.Layout != "" {
		comp = n.Layout
	}
	if comp == "" {
		comp = "div"
	}
	key := n.Key
	if key == "" {
		key = strings.Join(pathInts(path), "_")
	}
	var node *vdom.VNode
	native := map[string]bool{"div": true, "span": true, "input": true, "button": true, "textarea": true, "select": true, "form": true, "label": true, "p": true, "h1": true, "h2": true, "h3": true, "ul": true, "li": true, "a": true, "img": true}
	if native[comp] {
		node = vdom.NewElement(comp, key)
	} else {
		node = vdom.NewElement(comp, key)
	}
	if n.Animate != nil {
		node.Animate = &vdom.Animate{
			Type:      n.Animate.Type,
			Duration:  n.Animate.Duration,
			Easing:    n.Animate.Easing,
			Direction: n.Animate.Direction,
			Delay:     n.Animate.Delay,
		}
		node.WithProp("data-animate", n.Animate.Type)
		node.WithProp("data-animate-duration", fmt.Sprintf("%d", n.Animate.Duration))
		if n.Animate.Easing != "" {
			node.WithProp("data-animate-easing", n.Animate.Easing)
		}
		if n.Animate.Direction != "" {
			node.WithProp("data-animate-direction", n.Animate.Direction)
		}
	}
	for k, v := range n.Props {
		node.WithProp(k, e.resolveTextWithItem(v, item, index))
	}
	if n.Text != "" {
		node.WithText(e.resolveTextWithItem(n.Text, item, index))
	} else if n.Label != "" {
		node.WithText(e.resolveTextWithItem(n.Label, item, index))
	}
	if n.Bind != "" && n.Text == "" && n.Label == "" {
		if comp == "sl-input" || comp == "input" {
			if strings.HasPrefix(n.Bind, "item.") {
				field := strings.TrimPrefix(n.Bind, "item.")
				if v := getItemField(item, field); v != nil {
					node.WithProp("value", fmt.Sprintf("%v", v))
				}
			} else if n.Bind == "item" {
				node.WithProp("value", fmt.Sprintf("%v", item))
			} else {
				node.WithProp("value", e.state.GetString(n.Bind))
			}
		} else {
			if strings.HasPrefix(n.Bind, "item.") {
				field := strings.TrimPrefix(n.Bind, "item.")
				if v := getItemField(item, field); v != nil {
					node.WithText(fmt.Sprintf("%v", v))
				}
			} else if n.Bind == "item" {
				node.WithText(fmt.Sprintf("%v", item))
			} else {
				node.WithText(e.state.GetString(n.Bind))
			}
		}
	}
	for ev, act := range n.On {
		// подставляем index/item в действие если есть {{index}} / {{item.id}}
		resolvedAct := e.resolveTextWithItem(act, item, index)
		node.WithEvent(ev, resolvedAct)
	}
	for i, c := range n.Children {
		child := e.buildNodeWithItem(c, append(cloneInts(path), i), item, index)
		if child != nil {
			node.WithChild(child)
		}
	}
	return node
}

func pathInts(p []int) []string {
	out := make([]string, len(p))
	for i, v := range p {
		out[i] = itoa(v)
	}
	return out
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
