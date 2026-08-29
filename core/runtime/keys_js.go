//go:build js && wasm

package runtime

import (
	"strings"
	"syscall/js"
)

// BindKeys вешает обработчик keydown на document и выполняет действия
// из keys.yaml при совпадении сочетания.
func BindKeys(e *Engine) {
	if e.Keys() == nil || len(e.Keys().Bindings) == 0 {
		return
	}

	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		ev := args[0]
		meta := ev.Get("metaKey").Bool()
		ctrl := ev.Get("ctrlKey").Bool()
		shift := ev.Get("shiftKey").Bool()
		alt := ev.Get("altKey").Bool()

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

		key := strings.ToLower(ev.Get("key").String())
		if key == "" || key == " " {
			key = strings.ToLower(ev.Get("code").String())
		}

		b := e.findBinding(mods, key)
		if b == nil {
			return nil
		}
		if !e.shouldHandle(b) {
			return nil
		}
		// предотвращаем браузерное поведение по умолчанию.
		ev.Call("preventDefault")
		e.runBinding(b)
		return nil
	})

	js.Global().Get("document").Call("addEventListener", "keydown", fn)
}
