# Runtime API

## window.UIEngine

```ts
window.UIEngine.dispatch("inc state.counter.count")
window.UIEngine.getState("state.counter.count") // 5
window.UIEngine.setState("state.form.name", "hello")
window.UIEngine.registerComponent("myComp", { mount(container, props, onEvent) {} })
window.UIEngine.registerAction("myAction", () => {})
```

## ComponentHandle

```ts
{
  mount(container: HTMLElement, props: Record<string,string>, onEvent: (action: string)=>void): {update, unmount}
}
```

## Go

`core/runtime.Engine` — `Dispatch`, `SetScreen`, `RegisterAction`, `Subscribe`.

`core/state.Store` — `Set(path, value)`, `Get(path)`, `Subscribe(prefix, fn)`.

`core/dom.Renderer` — `Mount`, `Patch` с `data-animate` и `childNodes`.
