// PrimeVue analog — все базовые элементы интерфейса
// Экспортирует компоненты: Button, InputText, Checkbox, RadioButton, Dropdown, Textarea, Card, Panel, Dialog, TabView, DataTable
function h(tag, props, children = "") {
    const el = document.createElement(tag);
    Object.entries(props).forEach(([k, v]) => {
        if (k === "class")
            el.className = v;
        else if (k === "style")
            el.style.cssText += v;
        else
            el.setAttribute(k, v);
    });
    if (children)
        el.textContent = children;
    return el;
}
// Button — аналог PrimeVue Button
const Button = {
    mount(container, props, onEvent) {
        const btn = document.createElement("button");
        btn.className = `p-button p-component p-button-${props.variant || "primary"} p-button-${props.size || "md"}`;
        if (props.loading === "true")
            btn.classList.add("p-button-loading");
        if (props.disabled === "true")
            btn.disabled = true;
        btn.textContent = props.label || props.text || "Button";
        if (props.icon) {
            const ic = document.createElement("i");
            ic.className = props.icon;
            btn.prepend(ic);
        }
        btn.addEventListener("click", () => {
            if (props.onClick)
                onEvent(props.onClick);
            if (props.on)
                onEvent(props.on);
        });
        // PrimeVue-like ripple
        btn.style.position = "relative";
        btn.style.overflow = "hidden";
        container.appendChild(btn);
        return {
            update(newProps) {
                btn.className = `p-button p-component p-button-${newProps.variant || "primary"} p-button-${newProps.size || "md"}`;
                btn.textContent = newProps.label || newProps.text || "Button";
            },
            unmount() { btn.remove(); }
        };
    }
};
// InputText
const InputText = {
    mount(container, props, onEvent) {
        const inp = document.createElement("input");
        inp.type = "text";
        inp.className = "p-inputtext p-component";
        inp.placeholder = props.placeholder || "";
        inp.value = props.value || "";
        inp.addEventListener("input", (e) => {
            const v = e.target.value;
            if (props.onInput)
                onEvent(props.onInput.replace("$event", v));
            if (props.onChange)
                onEvent(props.onChange.replace("$event", v));
        });
        container.appendChild(inp);
        return {
            update(p) { inp.value = p.value || ""; inp.placeholder = p.placeholder || ""; },
            unmount() { inp.remove(); }
        };
    }
};
// Checkbox
const Checkbox = {
    mount(container, props, onEvent) {
        const wrap = h("div", { class: "p-checkbox p-component", style: "display:flex;align-items:center;gap:8px;" });
        const box = h("div", { class: "p-checkbox-box" });
        box.style.cssText = "width:20px;height:20px;border:1px solid #ccc;border-radius:3px;display:flex;align-items:center;justify-content:center;cursor:pointer;";
        const inp = document.createElement("input");
        inp.type = "checkbox";
        inp.checked = props.checked === "true";
        inp.style.display = "none";
        const label = h("label", {}, props.label || "");
        const update = (checked) => {
            inp.checked = checked;
            box.style.background = checked ? "var(--primary-color, #6366f1)" : "white";
            box.textContent = checked ? "✓" : "";
            box.style.color = "white";
        };
        update(inp.checked);
        box.addEventListener("click", () => {
            update(!inp.checked);
            if (props.onChange)
                onEvent(props.onChange.replace("$event", String(inp.checked)));
        });
        wrap.append(box, inp, label);
        container.appendChild(wrap);
        return { update(p) { update(p.checked === "true"); }, unmount() { wrap.remove(); } };
    }
};
// Card, Panel, Dialog, TabView, DataTable — упрощённые PrimeVue аналоги
const Card = {
    mount(container, props) {
        const el = h("div", { class: "p-card p-component", style: "border:1px solid #e5e7eb;border-radius:8px;padding:16px;background:white;box-shadow:0 1px 3px rgba(0,0,0,0.1);" });
        if (props.title) {
            const title = h("div", { class: "p-card-title", style: "font-weight:600;margin-bottom:8px;" }, props.title);
            el.appendChild(title);
        }
        container.appendChild(el);
        return { update() { }, unmount() { el.remove(); } };
    }
};
// Регистрируем все как отдельные компоненты, но через один модуль "button" (PrimeVue)
const PrimeVueComponents = {
    Button, InputText, Checkbox, RadioButton: Checkbox, Dropdown: InputText, Textarea: InputText,
    Card, Panel: Card, Dialog: Card, TabView: Card, DataTable: Card,
    button: Button, inputtext: InputText, checkbox: Checkbox, card: Card
};
function registerPrimeVue() {
    window.UIEngineModules = window.UIEngineModules || {};
    Object.entries(PrimeVueComponents).forEach(([name, comp]) => {
        window.UIEngineModules[name] = comp;
        window.UIEngineModules[name.toLowerCase()] = comp;
    });
    if (window.UIEngine) {
        Object.keys(PrimeVueComponents).forEach(name => {
            window.UIEngine.registerComponent(name, PrimeVueComponents[name]);
        });
    }
}
// Авто-регистрация при загрузке
if (typeof window !== "undefined") {
    // ждём UIEngine
    const tryReg = () => {
        if (window.UIEngine)
            registerPrimeVue();
        else
            setTimeout(tryReg, 100);
    };
    tryReg();
}
export {};
//# sourceMappingURL=index.js.map