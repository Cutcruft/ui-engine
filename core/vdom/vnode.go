// Package vdom содержит виртуальное DOM-дерево, из которого Go-ядро
// рендерит интерфейс в браузерный DOM через syscall/js.
//
// VNode — иммутабельное описание узла. Каждый рендер строит новое дерево,
// которое diff-ит с предыдущим (см. пакет diff) и применяет минимальные патчи.
package vdom

// Animate — декларативная анимация
type Animate struct {
	Type      string
	Duration  int
	Easing    string
	Direction string
	Delay     int
}

// VNode — виртуальный узел DOM.
type VNode struct {
	// Type — имя тега или Web Component (например "div", "sl-button").
	Type string
	// Key — стабильный ключ для ключевого диффа списков (аналог React key).
	Key string
	// Props — атрибуты/свойства узла.
	Props map[string]string
	// Events — обработчики событий: имя события -> ключ действия.
	Events map[string]string
	// Children — дочерние VNode или текстовые строки.
	Children []*VNode
	// Text — простой текстовый узел (Type == "#text").
	Text string
	// IsText — true, если это текстовый узел.
	IsText bool
	// Animate — анимация появления/исчезновения/перехода
	Animate *Animate
}

// NewElement создаёт элемент узла.
func NewElement(typ, key string) *VNode {
	return &VNode{Type: typ, Key: key, Props: map[string]string{}, Events: map[string]string{}}
}

// NewText создаёт текстовый узел.
func NewText(t string) *VNode {
	return &VNode{Type: "#text", Text: t, IsText: true, Props: map[string]string{}}
}

// NewShoelace создаёт элемент Web Component из стандартной библиотеки.
// Тег вида sl-<name>, ключ — стабильный идентификатор.
func NewShoelace(name, key string) *VNode {
	return NewElement("sl-"+name, key)
}

// WithProp добавляет атрибут/свойство.
func (n *VNode) WithProp(k, v string) *VNode {
	n.Props[k] = v
	return n
}

// WithText добавляет текстовый потомок.
func (n *VNode) WithText(t string) *VNode {
	n.Children = append(n.Children, NewText(t))
	return n
}

// WithChild добавляет дочерний узел.
func (n *VNode) WithChild(c *VNode) *VNode {
	n.Children = append(n.Children, c)
	return n
}

// WithEvent вешает обработчик события (имя -> ключ действия в runtime).
func (n *VNode) WithEvent(name, actionKey string) *VNode {
	n.Events[name] = actionKey
	return n
}
