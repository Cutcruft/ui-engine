package runtime

import (
	"testing"

	"github.com/ui-engine/core/config"
	"github.com/ui-engine/core/diff"
	"github.com/ui-engine/core/state"
	"github.com/ui-engine/core/vdom"
)

// mockRenderer собирает вызовы mount/patch для проверки диффа.
type mockRenderer struct {
	mounts  int
	patches [][]diff.Op
}

func (m *mockRenderer) Mount(_ *vdom.VNode) { m.mounts++ }
func (m *mockRenderer) Patch(ops []diff.Op) { m.patches = append(m.patches, ops) }

func newEngine() (*Engine, *state.Store, *mockRenderer) {
	st := state.New()
	app := &config.App{Name: "t", Root: "main"}
	// корень: column-layout с двумя детьми; второй — bind-узел на state.counter.
	root := &config.ScreenNode{
		Layout:   "column",
		Children: []*config.ScreenNode{
			{Component: "div", Text: "static"},
			{Component: "div", Bind: "state.counter"},
		},
	}
	screens := map[string]*config.Screen{
		"main": {Name: "main", Layout: root},
	}
	eng := NewEngine(app, screens, st)
	r := &mockRenderer{}
	eng.SetRenderer(r)
	return eng, st, r
}

func TestHookSetBoolAndTemplateRender(t *testing.T) {
	eng, st, r := newEngine()

	// узел-шаблон со хот счётчиком и булевой подписью.
	eng.screens["main"].Layout = &config.ScreenNode{
		Layout: "column",
		Children: []*config.ScreenNode{
			{Component: "span", Text: "знач = {{state.ui.ready}}"},
		},
	}
	eng.Mount()
	if r.mounts != 1 {
		t.Fatalf("expected 1 mount, got %d", r.mounts)
	}

	// Хук-действие из onReady. Set должен записать bool true.
	eng.RunHooks([]string{"set state.ui.ready = true"})

	v, ok := st.Get("state.ui.ready")
	if !ok {
		t.Fatal("state.ui.ready не создан")
	}
	if v != true {
		t.Fatalf("expected true, got %#v (%T)", v, v)
	}

	// Рендер шаблона должен показывать true.
	root := eng.Render()
	if len(root.Children) != 1 || len(root.Children[0].Children) != 1 {
		t.Fatalf("неожиданная структура корня: %+v", root)
	}
	got := root.Children[0].Children[0].Text
	if got != "знач = true" {
		t.Fatalf("expected 'знач = true', got %q", got)
	}
}

func TestHookSetBoolThroughSubscription(t *testing.T) {
	eng, st, r := newEngine()

	// зеркалим реальный main.yaml: hero (sl-card) с hero-body и отдельный statusbar
	eng.screens["main"].Layout = &config.ScreenNode{
		Component: "div",
		Children: []*config.ScreenNode{
			{Component: "sl-card", Children: []*config.ScreenNode{
				{Component: "div", Children: []*config.ScreenNode{
					{Component: "span", Text: "Счётчик: {{state.counter.count}}"},
				}},
			}},
			{Component: "div", Children: []*config.ScreenNode{
				{Component: "span", Text: "Состояние приложения: готово = {{state.ui.ready}}"},
			}},
		},
	}
	eng.Mount()

	// как в wasm/main.go: глобальная подписка state -> Apply
	st.Subscribe("state", func() { eng.Apply() })

	// как в RunHooks(onReady): set-хук ДО inc-хука (порядок из hooks.yaml)
	eng.RunHooks([]string{"set state.ui.ready = true", "inc state.counter.count"})

	if v, _ := st.Get("state.counter.count"); v != int64(1) && v != 1 {
		t.Fatalf("counter expected 1, got %#v", v)
	}
	// Проверяем, что Apply через подписку дал точечные патчи (а не через Render)
	if len(r.patches) == 0 {
		t.Fatalf("expected patches from subscription Apply, got 0 (mounts=%d)", r.mounts)
	}
	// Должны быть патчи для готово и счётчика (порядок: set -> inc, 2 патча)
	foundReady, foundCounter := false, false
	for _, ops := range r.patches {
		for _, op := range ops {
			if op.Kind == diff.OpSetText {
				// debug: выводим путь для диагностики
				// fmt.Printf("patch text=%q path=%v\n", op.Text, op.Path)
				_ = op.Path
			}
			if op.Kind == diff.OpSetText && op.Text == "Состояние приложения: готово = true" {
				foundReady = true
			}
			if op.Kind == diff.OpSetText && op.Text == "Счётчик: 1" {
				foundCounter = true
			}
		}
	}
	if !foundReady {
		t.Fatalf("ready patch not found in %+v (patches=%d)", r.patches, len(r.patches))
	}
	if !foundCounter {
		t.Fatalf("counter patch not found in %+v", r.patches)
	}
	root := eng.Render()
	status := root.Children[1]
	readyText := status.Children[0].Children[0].Text
	if readyText != "Состояние приложения: готово = true" {
		t.Fatalf("status expected 'готово = true', got %q (patches=%d)", readyText, len(r.patches))
	}
}

func TestReactivePartialDiff(t *testing.T) {
	eng, st, r := newEngine()

	eng.Mount()
	if r.mounts != 1 {
		t.Fatalf("expected 1 mount, got %d", r.mounts)
	}
	if len(r.patches) != 0 {
		t.Fatalf("expected no patches on mount, got %d", len(r.patches))
	}

	// Изменяем bind-путь состояния и применяем (как это делает глобальная
	// подписка "state" в wasm/main.go -> eng.Apply()).
	st.Set("state.counter", "5")
	eng.Apply()

	// Точечный пере-рендер: должен быть ровно один патч с OpSetText
	// по пути [1 0] (bind-узел — второй ребёнок корня, его текстовый узел),
	// без полного remount.
	if len(r.patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(r.patches))
	}
	ops := r.patches[0]
	if len(ops) != 1 || ops[0].Kind != diff.OpSetText {
		t.Fatalf("expected single OpSetText, got %+v", ops)
	}
	if len(ops[0].Path) != 2 || ops[0].Path[0] != 1 || ops[0].Path[1] != 0 {
		t.Fatalf("expected path [1 0], got %v", ops[0].Path)
	}
	if ops[0].Text != "5" {
		t.Fatalf("expected text '5', got %q", ops[0].Text)
	}
	// Никаких дополнительных remount'ов.
	if r.mounts != 1 {
		t.Fatalf("expected still 1 mount (no full remount), got %d", r.mounts)
	}
}

func TestReactiveUnknownPathNoRemount(t *testing.T) {
	eng, st, r := newEngine()
	eng.Mount()
	r.patches = nil

	// Не-реактивное изменение (никто не подписан на state.other) НЕ должно
	// приводить к полному remount; полный diff даёт 0 ops -> Patch не вызывается.
	st.Set("state.other", "x")
	eng.Apply()
	if r.mounts != 1 {
		t.Fatalf("expected no remount on non-reactive change, got mounts=%d", r.mounts)
	}
	if len(r.patches) != 0 {
		t.Fatalf("expected no patch batch for empty diff, got %v", r.patches)
	}
}

func TestReactiveSubscribeOnlyOnBind(t *testing.T) {
	// Проверяем, что реактивный узел реально подписан: изменённый префикс
	// = bind-путь -> dirty -> реактивный (а не полный) путь.
	eng, st, r := newEngine()
	eng.Mount()
	r.patches = nil

	st.Set("state.counter", "42")
	eng.Apply()

	ops := r.patches[len(r.patches)-1]
	if len(ops) == 0 || ops[0].Kind != diff.OpSetText {
		t.Fatalf("expected reactive OpSetText, got %+v", ops)
	}
}
