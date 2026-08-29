package vdom

// RenderFunc — функция, которая по контексту (состояние, theme) строит VNode.
type RenderFunc func(ctx *RenderContext) *VNode

// RenderContext передаётся в Render-функции экранов/компонентов.
type RenderContext struct {
	// State — доступ к реактивному состоянию (интерфейс, реализует state.Store).
	State StateReader
	// ResolveComponent резолвит имя компонента макета в RenderFunc.
	// Используется runtime-ем для подключения стандартных (Shoelace) и пользовательских модулей.
	ResolveComponent func(name string) (RenderFunc, bool)
}

// StateReader — минимальный интерфейс чтения состояния для рендера.
type StateReader interface {
	// Get возвращает значение по пути (например "state.form.login.email").
	// ok=false если пути нет.
	Get(path string) (any, bool)
}
