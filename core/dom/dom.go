//go:build js && wasm

// Package dom рендерит VDOM в браузерный DOM через syscall/js и применяет
// патчи из пакета diff.
package dom

import (
	"strings"
	"syscall/js"

	"github.com/ui-engine/core/diff"
	"github.com/ui-engine/core/vdom"
)

// Handler — функция, вызываемая при событии DOM. Получает key действия и
// (опционально) затронутое состояние/значение.
type Handler func(actionKey string)

// Renderer применяет VDOM-дерево к DOM-контейнеру.
type Renderer struct {
	// rootJS — JS-объект контейнера (например document.getElementById).
	rootJS js.Value
	// screenEl — смонтированный корневой экран (первый child rootJS после Mount).
	screenEl js.Value
	onEvent  Handler
}

// NewRenderer создаёт рендерер, монтированный в элемент с данным id.
func NewRenderer(containerID string, onEvent Handler) (*Renderer, error) {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", containerID)
	if !el.Truthy() {
		root := doc.Call("createElement", "div")
		root.Set("id", containerID)
		doc.Get("body").Call("appendChild", root)
		el = root
	}
	return &Renderer{rootJS: el, onEvent: onEvent}, nil
}

// Mount создаёт корневой VNode (по сути корень контейнера).
func (r *Renderer) Mount(root *vdom.VNode) {
	r.clear()
	r.applyCreate(r.rootJS, root, []int{})
	// запоминаем экранный корень для патчей (пути из Diff относительно него)
	r.screenEl = r.rootJS.Get("firstElementChild")
	if !r.screenEl.Truthy() {
		r.screenEl = r.rootJS
	}
}

// Patch применяет патчи относительно текущего состояния.
func (r *Renderer) Patch(ops []diff.Op) {
	doc := js.Global().Get("document")
	for _, op := range ops {
		switch op.Kind {
		case diff.OpCreate:
			if len(op.Path) == 0 {
				r.applyCreate(r.screenEl, op.Node, op.Path)
			} else {
				parentPath := op.Path[:len(op.Path)-1]
				idx := op.Path[len(op.Path)-1]
				if parent, ok := r.elementByPath(doc, parentPath); ok {
					// вставляем в конкретную позицию, а не всегда в конец
					children := parent.Get("childNodes")
					if children.Truthy() && idx < children.Get("length").Int() {
						ref := children.Index(idx)
						el := r.createElement(doc, op.Node)
						if ref.Truthy() {
							parent.Call("insertBefore", el, ref)
						} else {
							parent.Call("appendChild", el)
						}
					} else {
						r.applyCreate(parent, op.Node, op.Path)
					}
				} else {
					r.applyCreate(r.screenEl, op.Node, op.Path)
				}
			}
		case diff.OpRemove:
			if el, ok := r.elementAt(doc, op.Path); ok {
				// текст-узлы не анимируем
				if el.Get("nodeType").Int() == 3 {
					el.Call("remove")
					continue
				}
				// leave animation if data-animate present
				animType := ""
				if fn := el.Get("getAttribute"); fn.Truthy() {
					if v := el.Call("getAttribute", "data-animate"); v.Truthy() {
						animType = v.String()
					}
				}
				if animType != "" && animType != "none" {
					durStr := el.Call("getAttribute", "data-animate-duration")
					dur := 200
					if durStr.Truthy() {
						if n, err := parseInt(durStr.String()); err == nil {
							dur = n
						}
					}
					// apply leave: fade/slide/scale out
					el.Get("style").Set("transition", "opacity "+itoa(dur)+"ms ease, transform "+itoa(dur)+"ms ease")
					switch animType {
					case "fade":
						el.Get("style").Set("opacity", "0")
					case "slide":
						el.Get("style").Set("opacity", "0")
						el.Get("style").Set("transform", "translateY(-10px)")
					case "scale":
						el.Get("style").Set("opacity", "0")
						el.Get("style").Set("transform", "scale(0.95)")
					default:
						el.Get("style").Set("opacity", "0")
					}
					// delay remove
					elCopy := el
					js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
						elCopy.Call("remove")
						return nil
					}), dur)
				} else {
					el.Call("remove")
				}
			} else {
				println("Patch OpRemove navigate fail", len(op.Path))
			}
		case diff.OpSetText:
			if el, ok := r.elementAt(doc, op.Path); ok {
				el.Set("textContent", op.Text)
			} else {
				println("Patch OpSetText FAIL path", len(op.Path))
				for _, p := range op.Path {
					println(p)
				}
			}
		case diff.OpSetProp:
			if el, ok := r.nodeAt(doc, op.Path); ok {
				applyProp(el, op)
			}
		case diff.OpRemoveProp:
			if el, ok := r.nodeAt(doc, op.Path); ok {
				removeProp(el, op.Prop)
			}
		}
	}
}

// navigate возвращает элемент по пути индексов (childNodes). Если путь пустой — экранный корень.
func (r *Renderer) navigate(doc js.Value, path []int) (js.Value, bool) {
	if len(path) == 0 {
		if r.screenEl.Truthy() {
			return r.screenEl, true
		}
		return r.rootJS, true
	}
	return r.elementByPath(doc, path)
}

// elementByPath идёт по childNodes, начиная с экранного корня (чтобы путь совпадал
// с индексами VNode Children — пути из Diff относительно screen root).
func (r *Renderer) elementByPath(doc js.Value, path []int) (js.Value, bool) {
	base := r.screenEl
	if !base.Truthy() {
		base = r.rootJS
	}
	cur := base
	for _, idx := range path {
		if idx < 0 || !cur.Truthy() {
			return js.Undefined(), false
		}
		n := cur.Get("childNodes")
		if !n.Truthy() {
			return js.Undefined(), false
		}
		if idx >= n.Get("length").Int() {
			return js.Undefined(), false
		}
		cur = n.Index(idx)
		if !cur.Truthy() {
			return js.Undefined(), false
		}
	}
	return cur, true
}

// nodeAt — то же что elementByPath (для свойств/удаления).
func (r *Renderer) nodeAt(doc js.Value, path []int) (js.Value, bool) {
	return r.elementByPath(doc, path)
}

// elementAt — поиск элемента для текстового узла.
func (r *Renderer) elementAt(doc js.Value, path []int) (js.Value, bool) {
	return r.elementByPath(doc, path)
}

var nativeTags = map[string]bool{
	"div": true, "span": true, "p": true, "h1": true, "h2": true, "h3": true,
	"ul": true, "li": true, "a": true, "img": true, "form": true, "label": true,
	"sl-button": true, "sl-card": true, "sl-alert": true, "sl-input": true,
}

func isPluginComponent(typ string) bool {
	if nativeTags[typ] {
		return false
	}
	// sl-* are considered native for now (Shoelace), but could be plugin
	if strings.HasPrefix(typ, "sl-") {
		return false
	}
	return true
}

func (r *Renderer) createElement(doc js.Value, node *vdom.VNode) js.Value {
	if node.IsText {
		return doc.Call("createTextNode", node.Text)
	}
	// plugin component — делегируем в JS (window.UIEngine.getComponent или window.UIEngineModules)
	if isPluginComponent(node.Type) {
		println("tryMountPlugin", node.Type)
		if el := r.tryMountPlugin(doc, node); el.Truthy() {
			println("tryMountPlugin success", node.Type)
			return el
		}
		println("tryMountPlugin fallback", node.Type)
		// fallback — обычный div с data-component
	}
	el := doc.Call("createElement", node.Type)
	// для неизвестных компонентов (plugin) создаём div с data-component
	if isPluginComponent(node.Type) {
		// уже обработано выше, но fallback
		el.Call("setAttribute", "data-component", node.Type)
	}
	for k, v := range node.Props {
		applyRawProp(el, k, v)
	}
	for k, v := range node.Events {
		attachEvent(el, k, v, r.onEvent)
	}
	for _, c := range node.Children {
		r.appendInto(doc, el, c)
	}
	// enter animation
	if node.Animate != nil && node.Animate.Type != "" && node.Animate.Type != "none" {
		applyEnterAnimation(el, node.Animate)
	} else if v, ok := node.Props["data-animate"]; ok && v != "" && v != "none" {
		dur := 200
		if d, ok := node.Props["data-animate-duration"]; ok {
			if n, err := parseInt(d); err == nil {
				dur = n
			}
		}
		applyEnterAnimation(el, &vdom.Animate{Type: v, Duration: dur, Easing: node.Props["data-animate-easing"]})
	}
	return el
}

func (r *Renderer) tryMountPlugin(doc js.Value, node *vdom.VNode) js.Value {
	// проверяем window.UIEngineModules[name] или window.UIEngine.getComponent
	var mod js.Value
	if modules := js.Global().Get("UIEngineModules"); modules.Truthy() {
		mod = modules.Get(node.Type)
	}
	if !mod.Truthy() {
		if uiEngine := js.Global().Get("UIEngine"); uiEngine.Truthy() {
			if fn := uiEngine.Get("getComponent"); fn.Truthy() {
				mod = fn.Invoke(node.Type)
			}
		}
	}
	if !mod.Truthy() || mod.IsNull() || mod.IsUndefined() {
		return js.Null()
	}
	// mod should have mount(container, props, onEvent)
	// создаём контейнер для плагина
	container := doc.Call("createElement", "div")
	container.Call("setAttribute", "data-plugin", node.Type)
	container.Call("setAttribute", "data-key", node.Key)
	// собираем props в JS объект
	propsObj := js.Global().Get("Object").New()
	for k, v := range node.Props {
		propsObj.Set(k, v)
	}
	// onEvent wrapper
	onEvent := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			action := args[0].String()
			if r.onEvent != nil {
				r.onEvent(action)
			}
		}
		return nil
	})
	// вызываем mount
	if mountFn := mod.Get("mount"); mountFn.Truthy() {
		mountFn.Invoke(container, propsObj, onEvent)
		// также пробрасываем children как text (для простых случаев)
		for _, c := range node.Children {
			if c.IsText {
				container.Set("textContent", c.Text)
			} else {
				// для сложных children — рекурсивно
				childEl := r.createElement(doc, c)
				container.Call("appendChild", childEl)
			}
		}
		// enter animation for plugin container
		if node.Animate != nil {
			applyEnterAnimation(container, node.Animate)
		}
		return container
	}
	return js.Null()
}

func resolveDurationToken(s string) int {
	// поддержка токенов из theme.animations
	switch s {
	case "fast", "durationFast":
		return 150
	case "normal", "durationNormal":
		return 300
	case "slow", "durationSlow":
		return 500
	}
	if n, err := parseInt(s); err == nil {
		return n
	}
	return 0
}

func resolveEasingToken(s string) string {
	switch s {
	case "spring", "easingSpring":
		return "cubic-bezier(0.34, 1.56, 0.64, 1)"
	case "easeOut", "easingEaseOut":
		return "cubic-bezier(0, 0, 0.2, 1)"
	case "easeInOut":
		return "cubic-bezier(0.4, 0, 0.2, 1)"
	}
	return s
}

func applyEnterAnimation(el js.Value, a *vdom.Animate) {
	if a == nil || a.Type == "" || a.Type == "none" {
		return
	}
	dur := a.Duration
	if dur == 0 {
		// пробуем токен
		if d := resolveDurationToken(a.Easing); d != 0 && a.Type == "spring" {
			dur = 400
		} else {
			dur = 200
		}
	}
	easing := a.Easing
	if easing == "" {
		easing = "ease"
	}
	easing = resolveEasingToken(easing)
	// поддержка spring как отдельный тип
	if a.Type == "spring" {
		// spring physics via transform
		el.Get("style").Set("transition", "none")
		el.Get("style").Set("opacity", "0")
		el.Get("style").Set("transform", "scale(0.96) translateY(8px)")
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			el.Get("style").Set("transition", "opacity 400ms "+easing+", transform 400ms "+easing)
			el.Get("style").Set("opacity", "1")
			el.Get("style").Set("transform", "scale(1) translateY(0)")
			return nil
		}), 10)
		return
	}
	// set transition
	el.Get("style").Set("transition", parseTransition(a))
	// initial state
	switch a.Type {
	case "fade":
		el.Get("style").Set("opacity", "0")
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			el.Get("style").Set("opacity", "1")
			return nil
		}), 10)
	case "slide":
		dir := a.Direction
		if dir == "" {
			dir = "up"
		}
		var tx string
		switch dir {
		case "left":
			tx = "translateX(-20px)"
		case "right":
			tx = "translateX(20px)"
		case "up":
			tx = "translateY(-12px)"
		case "down":
			tx = "translateY(12px)"
		default:
			tx = "translateY(-12px)"
		}
		el.Get("style").Set("opacity", "0")
		el.Get("style").Set("transform", tx)
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			el.Get("style").Set("opacity", "1")
			el.Get("style").Set("transform", "translateX(0) translateY(0)")
			return nil
		}), 10)
	case "scale":
		el.Get("style").Set("opacity", "0")
		el.Get("style").Set("transform", "scale(0.95)")
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			el.Get("style").Set("opacity", "1")
			el.Get("style").Set("transform", "scale(1)")
			return nil
		}), 10)
	}
	_ = dur
	_ = easing
}

func parseTransition(a *vdom.Animate) string {
	dur := a.Duration
	if dur == 0 {
		dur = 200
	}
	easing := a.Easing
	if easing == "" {
		easing = "ease"
	}
	switch a.Type {
	case "fade":
		return "opacity " + itoa(dur) + "ms " + easing
	case "slide":
		return "opacity " + itoa(dur) + "ms " + easing + ", transform " + itoa(dur) + "ms " + easing
	case "scale":
		return "opacity " + itoa(dur) + "ms " + easing + ", transform " + itoa(dur) + "ms " + easing
	default:
		return "all " + itoa(dur) + "ms " + easing
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, js.Error{Value: js.ValueOf("not int")}
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func (r *Renderer) applyCreate(parent js.Value, node *vdom.VNode, path []int) {
	if node == nil {
		return
	}
	doc := js.Global().Get("document")
	el := r.createElement(doc, node)
	parent.Call("appendChild", el)
}

func (r *Renderer) appendInto(doc js.Value, parent js.Value, node *vdom.VNode) {
	if node.IsText {
		parent.Call("appendChild", doc.Call("createTextNode", node.Text))
		return
	}
	el := doc.Call("createElement", node.Type)
	for k, v := range node.Props {
		applyRawProp(el, k, v)
	}
	for k, v := range node.Events {
		attachEvent(el, k, v, r.onEvent)
	}
	for _, c := range node.Children {
		r.appendInto(doc, el, c)
	}
	if node.Animate != nil && node.Animate.Type != "" && node.Animate.Type != "none" {
		applyEnterAnimation(el, node.Animate)
	}
	parent.Call("appendChild", el)
}

func (r *Renderer) clear() {
	r.rootJS.Set("innerHTML", "")
}

// applyProp обрабатывает атрибуты и события при патче OpSetProp.
func applyProp(el js.Value, op diff.Op) {
	if stringsHasPrefix(op.Prop, "on:") {
		// в MVP события не снимаются при patch; пере-назначаем
		eventName := op.Prop[3:]
		attachEvent(el, eventName, op.Value, nil) // nil -> обработаем в render через capture
		return
	}
	applyRawProp(el, op.Prop, op.Value)
}

func applyRawProp(el js.Value, name, val string) {
	switch name {
	case "textContent", "innerHTML":
		el.Set(name, val)
	case "className":
		el.Set("className", val)
	case "value", "checked", "href", "src", "disabled", "variant", "size", "type", "contenteditable", "contentEditable":
		el.Set(name, val)
		// также атрибут для contenteditable
		if name == "contenteditable" || name == "contentEditable" {
			el.Call("setAttribute", "contenteditable", val)
		}
	default:
		el.Call("setAttribute", name, val)
	}
}

func removeProp(el js.Value, name string) {
	el.Call("removeAttribute", name)
}

// attachEvent вешает обработчик. Поддерживает подстановку $event в action.
func attachEvent(el js.Value, eventName, actionKey string, h Handler) {
	if h == nil {
		return
	}
	key := actionKey
	el.Call("addEventListener", eventName, js.FuncOf(func(this js.Value, args []js.Value) any {
		k := key
		if strings.Contains(k, "$event") {
			var evVal string
			if len(args) > 0 {
				ev := args[0]
				// пробуем event.target.value (для input/sl-input)
				if target := ev.Get("target"); target.Truthy() {
					if v := target.Get("value"); v.Truthy() {
						evVal = v.String()
					}
				}
				// sl-input может хранить value в detail
				if evVal == "" {
					if detail := ev.Get("detail"); detail.Truthy() {
						if v := detail.Get("value"); v.Truthy() {
							evVal = v.String()
						}
					}
				}
				// fallback: this.value
				if evVal == "" && this.Truthy() {
					if v := this.Get("value"); v.Truthy() {
						evVal = v.String()
					}
				}
				// contenteditable (richtext) -> innerHTML
				if evVal == "" {
					if target := ev.Get("target"); target.Truthy() {
						if ce := target.Get("isContentEditable"); ce.Truthy() && ce.Bool() {
							if v := target.Get("innerHTML"); v.Truthy() {
								evVal = v.String()
							}
						}
					}
					if evVal == "" && this.Truthy() {
						if ce := this.Get("isContentEditable"); ce.Truthy() && ce.Bool() {
							if v := this.Get("innerHTML"); v.Truthy() {
								evVal = v.String()
							}
						}
					}
				}
			}
			k = strings.ReplaceAll(k, "$event", evVal)
		}
		if h != nil {
			h(k)
		}
		return nil
	}))
}

func stringsHasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
