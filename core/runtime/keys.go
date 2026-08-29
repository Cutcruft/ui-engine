package runtime

import (
	"strings"

	"github.com/ui-engine/core/config"
)

// parseCombo разбирает строку сочетания ("meta+s", "shift+/") в набор
// признаков: набор модификаторов + основной ключ.
func parseCombo(combo string) (mods map[string]bool, key string) {
	mods = map[string]bool{}
	parts := strings.Split(combo, "+")
	last := parts[len(parts)-1]
	for _, p := range parts[:len(parts)-1] {
		switch strings.ToLower(p) {
		case "meta", "cmd", "command":
			mods["meta"] = true
		case "ctrl", "control":
			mods["ctrl"] = true
		case "shift":
			mods["shift"] = true
		case "alt", "option":
			mods["alt"] = true
		}
	}
	return mods, strings.ToLower(last)
}

// matchCombo проверяет, совпадает ли набор модификаторов и ключ.
func matchCombo(bindingKey string, mods map[string]bool, key string) bool {
	needMods, needKey := parseCombo(bindingKey)
	if needKey != key {
		return false
	}
	for m := range needMods {
		if !mods[m] {
			return false
		}
	}
	// лишние модификаторы не решают (например meta+ctrl+s не триггерит meta+s)
	return true
}

// keyName извлекает имя клавиши из события keydown.
func keyName(meta, ctrl, shift, alt bool, code string) string {
	mods := map[string]bool{}
	if meta {
		mods["meta"] = true
	}
	if ctrl {
		mods["ctrl"] = true
	}
	if shift {
		mods["shift"] = true
	}
	if alt {
		mods["alt"] = true
	}
	_ = mods
	// Полная проверка матчинга делается в genericKeys (см. ниже),
	// здесь просто возвращаем code в нижнем регистре.
	return strings.ToLower(code)
}

// findBinding ищет назначение, чьё комбо совпадает с модификаторами+ключом.
func (e *Engine) findBinding(mods map[string]bool, key string) *config.KeyBinding {
	if e.keys == nil {
		return nil
	}
	for _, kb := range e.keys.Bindings {
		for _, combo := range kb.Combo {
			if matchCombo(combo, mods, key) {
				return &kb
			}
		}
	}
	return nil
}

// ShouldHandleCheck — условие when для привязки (путь состояния truthy).
func (e *Engine) shouldHandle(kb *config.KeyBinding) bool {
	if kb.When == "" {
		return true
	}
	if v, ok := e.state.Get(kb.When); ok {
		return !isFalsy(v)
	}
	return false
}

// runBinding выполняет действие, заданное клавишей.
func (e *Engine) runBinding(kb *config.KeyBinding) {
	if kb.Action == "" {
		return
	}
	// если у действия указаны аргументы, пробуем их применить (только set вида arg=val)
	if len(kb.Args) > 0 {
		for _, a := range kb.Args {
			e.argApply(a)
		}
	}
	e.Dispatch(kb.Action)
}

// argApply применяет аргумент вида "path=value" к состоянию.
func (e *Engine) argApply(arg string) {
	eq := strings.Index(arg, "=")
	if eq < 0 {
		return
	}
	path := strings.TrimSpace(arg[:eq])
	val := strings.TrimSpace(arg[eq+1:])
	if path != "" {
		e.state.Set(path, parseValue(val))
	}
}
