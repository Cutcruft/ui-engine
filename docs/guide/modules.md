# Модули

Внешние плагины, не часть ядра. Ядро не знает о `button`/`row`/`richtext` — делегирует в `window.UIEngineModules`.

## Создание

```sh
ui-engine module new my-mod --type component --ts --with-tests --with-docs --with-example
# -> my-mod/module.yaml, src/index.ts, ui/component.yaml, docs/, test.js
```

`module.yaml`:

```yaml
module:
  name: my-mod
  version: 0.1.0
  type: component
  entry: {js: src/index.ts}
  components: [{name: my-mod, props: {variant: {type: string}}}]
```

`src/index.ts`:

```ts
import type { ComponentHandle } from "../../runtime-js/src/types";
const MyComp: ComponentHandle = {
  mount(container, props, onEvent) {
    const el = document.createElement("div");
    el.textContent = props.text || "";
    container.appendChild(el);
    return { update(p) { el.textContent = p.text; }, unmount() { el.remove(); } };
  }
};
(window as any).UIEngineModules["my-mod"] = MyComp;
```

## Подключение

```sh
ui-engine module add my-mod                    # из stdlib
ui-engine module add ./my-mod --local         # локально
ui-engine module add my-mod@1.2.0             # версия
# -> modules/my-mod/ + запись в app.yaml:modules
```

```yaml
# app.yaml
modules:
  - name: my-mod
    version: 0.1.0
    path: modules/my-mod
```

## Использование

```yaml
- component: my-mod
  props: {variant: primary}
  on: {click: "inc state.x"}
```

## Публикация

```sh
# .uimod — архив модуля
zip -r my-mod.uimod my-mod/
# registry пока локальный (stdlib), в будущем — remote
```
