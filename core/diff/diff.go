// Package diff сравнивает два VDOM-дерева и выдаёт список патчей
// для минимального изменения браузерного DOM.
package diff

import "github.com/ui-engine/core/vdom"

// Op — операция патча DOM.
type Op struct {
	Kind OpKind
	// Path — путь к узлу (индексы). Пустая строка = корень (как #root).
	Path []int
	// Prop — имя атрибута/свойства для OpSetProp/OpRemoveProp.
	Prop string
	// Value — значение для OpSetProp/OpSetText/OpCreate.
	Value string
	// Node — создаваемый узел для OpCreate.
	Node *vdom.VNode
	// Index — позиция вставки/удаления для дочерних операций.
	Index int
	// Text — для OpSetText.
	Text string
}

// OpKind — тип операции.
type OpKind int

const (
	OpCreate OpKind = iota
	OpRemove
	OpSetText
	OpSetProp
	OpRemoveProp
)

// Diff сравнивает old и new деревья (с корнем в Path{}) и возвращает список операций.
func Diff(old, new *vdom.VNode, path []int) []Op {
	ops := []Op{}
	if old == nil {
		ops = append(ops, Op{Kind: OpCreate, Path: clonePath(path), Node: new})
		return ops
	}
	if new == nil {
		ops = append(ops, Op{Kind: OpRemove, Path: clonePath(path)})
		return ops
	}

	// Тип узла изменился -> пересоздать.
	if old.Type != new.Type || old.IsText != new.IsText {
		ops = append(ops, Op{Kind: OpRemove, Path: clonePath(path)})
		ops = append(ops, Op{Kind: OpCreate, Path: clonePath(path), Node: new})
		return ops
	}

	// Текст изменился.
	if old.IsText && old.Text != new.Text {
		ops = append(ops, Op{Kind: OpSetText, Path: clonePath(path), Text: new.Text})
		return ops
	}

	// Свойства.
	for k, v := range new.Props {
		if old.Props[k] != v {
			ops = append(ops, Op{Kind: OpSetProp, Path: clonePath(path), Prop: k, Value: v})
		}
	}
	for k := range old.Props {
		if _, ok := new.Props[k]; !ok {
			ops = append(ops, Op{Kind: OpRemoveProp, Path: clonePath(path), Prop: k})
		}
	}

	// События — трактуем как атрибуты вида on:<event>=<action>.
	allProps := map[string]string{}
	for k, v := range new.Props {
		allProps[k] = v
	}
	for k, v := range new.Events {
		allProps["on:"+k] = v
	}
	oldProps := map[string]string{}
	for k, v := range old.Props {
		oldProps[k] = v
	}
	for k, v := range old.Events {
		oldProps["on:"+k] = v
	}
	for k, v := range allProps {
		if oldProps[k] != v {
			ops = append(ops, Op{Kind: OpSetProp, Path: clonePath(path), Prop: k, Value: v})
		}
	}
	for k := range oldProps {
		if _, ok := allProps[k]; !ok {
			ops = append(ops, Op{Kind: OpRemoveProp, Path: clonePath(path), Prop: k})
		}
	}

	// Дети — ключевой дифф.
	ops = append(ops, diffChildren(old.Children, new.Children, path)...)

	return ops
}

func diffChildren(oldCh, newCh []*vdom.VNode, base []int) []Op {
	ops := []Op{}

	// Маппинг ключей старого -> индекс.
	oldByKey := map[string]int{}
	for i, c := range oldCh {
		if c.Key != "" {
			oldByKey[c.Key] = i
		}
	}

	// Флаг: какой старый индекс уже сопоставлен (чтобы бесключевой позиционно
	// не "съел" ключевой узел).
	matched := make([]bool, len(oldCh))

	// Проход по новым узлам.
	for i, nc := range newCh {
		childPath := append(clonePath(base), i)

		if nc.Key != "" {
			// Узел с ключом: ищем в старом.
			if oi, ok := oldByKey[nc.Key]; ok {
				matched[oi] = true
				ops = append(ops, Diff(oldCh[oi], nc, append(clonePath(base), oi))...)
				continue
			}
			// Ключа нет в старом -> создать (при этом позиция может сдвинуться).
			ops = append(ops, Op{Kind: OpCreate, Path: childPath, Node: nc})
			continue
		}

		// Бесключевой узел: сопоставляем позиционно со старым без ключа.
		// Ищем первый несовпавший бесключевой старый индекс на позиции i или ближайший.
		oi := i
		if oi < len(oldCh) && !matched[oi] && oldCh[oi].Key == "" {
			matched[oi] = true
			ops = append(ops, Diff(oldCh[oi], nc, append(clonePath(base), oi))...)
			continue
		}
		// Иначе ищем следующий свободный бесключевой.
		found := -1
		for j := 0; j < len(oldCh); j++ {
			if !matched[j] && oldCh[j].Key == "" {
				found = j
				break
			}
		}
		if found >= 0 {
			matched[found] = true
			ops = append(ops, Diff(oldCh[found], nc, append(clonePath(base), found))...)
			continue
		}
		ops = append(ops, Op{Kind: OpCreate, Path: childPath, Node: nc})
	}

	// Удаляем несопоставленные старые узлы.
	for i := range oldCh {
		if !matched[i] {
			ops = append(ops, Op{Kind: OpRemove, Path: append(clonePath(base), i)})
		}
	}

	return ops
}

func clonePath(p []int) []int {
	if p == nil {
		return []int{}
	}
	c := make([]int, len(p))
	copy(c, p)
	return c
}
