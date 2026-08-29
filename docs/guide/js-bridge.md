# JS Bridge

Двусторонний мост Go ↔ JS.

## window.UIEngine (TS)

```ts
window.UIEngine.dispatch("inc state.counter.count")
window.UIEngine.getState("state.counter.count") // 5
window.UIEngine.setState("state.form.name", "hello")

window.UIEngine.registerComponent("myComp", {
  mount(container, props, onEvent) {
    const el = document.createElement("div");
    el.textContent = props.text;
    el.addEventListener("click", () => onEvent(props.onClick));
    container.appendChild(el);
    return { update(p) { el.textContent = p.text; }, unmount() { el.remove(); } };
  }
})

window.UIEngine.registerAction("jsHello", () => {
  console.log("from Go: action.jsHello");
  window.UIEngine.dispatch("inc state.counter.count");
})
```

```yaml
# YAML
- component: sl-button
  on: {click: "action.jsHello"}
```

Go регистрирует `action.*` через `eng.RegisterAction("action.jsHello", fn)` (`core/wasm/main.go:213`).

## События

- `state` → Go → `eng.Apply()` → `diff` → `Patch`
- `DOM click` → `attachEvent` (`dom.go:409` `$event` → `event.target.value`/`innerHTML`) → `Dispatch`
- `JS` → `dispatch` → Go

Песочница: `bridge.js` исполняется в `window` с доступом только к `window.UIEngine`.
