# Компоненты

Все компоненты — внешние плагины, ядро их не знает.

## Использование

```yaml
- component: Button
  props: {variant: primary, size: md}
  label: "Нажми"
  on: {click: "inc state.counter.count"}

- component: InputText
  bind: state.form.name
  props: {placeholder: "Имя"}
  on: {input: "set state.form.name = $event"}

- component: Card
  props: {title: "Карточка"}
  children:
    - component: span
      text: "Контент"
```

## PrimeVue (button)

Модуль `button` — аналог PrimeVue, все базовые элементы:

- `Button` — `variant: primary/secondary/ghost/danger`, `size: sm/md/lg`, `loading/disabled/pill/icon`
- `InputText`, `Textarea`, `Checkbox`, `RadioButton`, `Dropdown`, `Select`
- `Card`, `Panel`, `Dialog`, `TabView`, `DataTable`

Установка:
```sh
ui-engine module add button
```

## Richtext (tiptap)

```yaml
- component: richtext
  bind: state.doc.content
  props: {placeholder: "Начните...", toolbar: full}
  on: {change: "set state.doc.content = $event"}
```

Модуль `richtext` — notion-like: headings, bold, lists, todo, code, quote, link, image, table.

```sh
ui-engine module add richtext
```

## Кастомный компонент

```ts
// my-module/src/index.ts
window.UIEngine.registerComponent("myComp", {
  mount(container, props, onEvent) {
    const el = document.createElement("div");
    el.textContent = props.text || "";
    el.addEventListener("click", () => onEvent(props.onClick));
    container.appendChild(el);
    return { update(p) { el.textContent = p.text; }, unmount() { el.remove(); } };
  }
});
```
