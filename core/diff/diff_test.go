package diff

import (
	"testing"

	"github.com/ui-engine/core/vdom"
)

// toplevel() строит дерево:
// <div>
//   <sl-button>Рабочее</sl-button>
//   <input/>
// </div>
func tree1() *vdom.VNode {
	root := vdom.NewElement("div", "root")
	btn := vdom.NewShoelace("button", "b1").WithText("Рабочее")
	inp := vdom.NewElement("input", "i1").WithProp("placeholder", "Текст")
	root.WithChild(btn).WithChild(inp)
	return root
}

func tree2() *vdom.VNode {
	root := vdom.NewElement("div", "root")
	btn := vdom.NewShoelace("button", "b1").WithText("Обновлено")
	inp := vdom.NewElement("input", "i1").WithProp("placeholder", "Новый")
	root.WithChild(btn).WithChild(inp)
	return root
}

func TestDiffTextChange(t *testing.T) {
	ops := Diff(tree1(), tree2(), nil)
	// ожидаем: set text на кнопке + set prop placeholder
	foundText := false
	foundPlaceholder := false
	for _, op := range ops {
		if op.Kind == OpSetText && op.Text == "Обновлено" {
			foundText = true
		}
		if op.Kind == OpSetProp && op.Prop == "placeholder" && op.Value == "Новый" {
			foundPlaceholder = true
		}
	}
	if !foundText || !foundPlaceholder {
		t.Fatalf("expected text+placeholder updates, got %#v", ops)
	}
}

func TestDiffCreateNil(t *testing.T) {
	ops := Diff(nil, tree1(), nil)
	if len(ops) != 1 || ops[0].Kind != OpCreate {
		t.Fatalf("expected single create, got %#v", ops)
	}
}

func TestDiffTypeChangeCreates(t *testing.T) {
	a := vdom.NewElement("div", "x")
	b := vdom.NewElement("span", "x")
	ops := Diff(a, b, nil)
	if len(ops) != 2 {
		t.Fatalf("expected remove+create, got %#v", ops)
	}
}
