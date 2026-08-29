//go:build js && wasm

package net

import (
	"syscall/js"
)

// jsTransport — WebSocket-транспорт через браузерный WebSocket API.
type jsTransport struct {
	ws js.Value
}

// NewJSTransport создаёт WebSocket-транспорт для браузера.
func NewJSTransport() Transport { return &jsTransport{} }

func (t *jsTransport) Connect(url string, onOpen func(), onMessage func([]byte), onClose func()) error {
	wsCtor := js.Global().Get("WebSocket")
	if !wsCtor.Truthy() {
		// WebSocket API недоступен (например, SSR-тест) — считаем неоткрытым.
		go func() { onClose() }()
		return nil
	}
	ws := wsCtor.New(url)
	t.ws = ws

	// Колбэки через js.FuncOf.
	onOpenFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		onOpen()
		return nil
	})
	onMsgFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			data := args[0].Get("data")
			if data.Truthy() {
				onMessage([]byte(data.String()))
			}
		}
		return nil
	})
	onCloseFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		onClose()
		return nil
	})
	onErrFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		onClose()
		return nil
	})

	ws.Set("onopen", onOpenFn)
	ws.Set("onmessage", onMsgFn)
	ws.Set("onclose", onCloseFn)
	ws.Set("onerror", onErrFn)
	return nil
}

func (t *jsTransport) Send(data []byte) {
	if !t.ws.Truthy() {
		return
	}
	t.ws.Call("send", string(data))
}

func (t *jsTransport) Close() {
	if t.ws.Truthy() {
		t.ws.Call("close")
	}
}
