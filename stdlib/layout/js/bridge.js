// Vue Flex гибрид — утилиты PrimeFlex + компоненты Row/Col/Grid/Stack/Container
// Утилиты: p-d-flex, p-jc-between, p-ai-center, p-col-6, p-m-2 и т.д. — генерируются как CSS
// Компоненты: Row, Col, Grid, Stack, Container
const FlexUtilsCSS = `
.p-d-flex { display: flex; }
.p-d-none { display: none; }
.p-jc-start { justify-content: flex-start; }
.p-jc-end { justify-content: flex-end; }
.p-jc-center { justify-content: center; }
.p-jc-between { justify-content: space-between; }
.p-jc-around { justify-content: space-around; }
.p-ai-start { align-items: flex-start; }
.p-ai-center { align-items: center; }
.p-ai-end { align-items: flex-end; }
.p-ai-stretch { align-items: stretch; }
.p-flex-column { flex-direction: column; }
.p-flex-row { flex-direction: row; }
.p-flex-wrap { flex-wrap: wrap; }
.p-col-1 { flex: 0 0 8.3333%; max-width: 8.3333%; }
.p-col-2 { flex: 0 0 16.6667%; max-width: 16.6667%; }
.p-col-3 { flex: 0 0 25%; max-width: 25%; }
.p-col-4 { flex: 0 0 33.3333%; max-width: 33.3333%; }
.p-col-6 { flex: 0 0 50%; max-width: 50%; }
.p-col-12 { flex: 0 0 100%; max-width: 100%; }
.p-m-1 { margin: 0.25rem; }
.p-m-2 { margin: 0.5rem; }
.p-p-1 { padding: 0.25rem; }
.p-p-2 { padding: 0.5rem; }
.p-gap-1 { gap: 0.25rem; }
.p-gap-2 { gap: 0.5rem; }
`;
function injectFlexCSS() {
    if (document.querySelector("#primeflex-css"))
        return;
    const style = document.createElement("style");
    style.id = "primeflex-css";
    style.textContent = FlexUtilsCSS;
    document.head.appendChild(style);
}
function mkFlex(tag, baseStyle) {
    return {
        mount(container, props) {
            injectFlexCSS();
            const el = document.createElement(tag);
            el.className = props.class || "";
            // утилиты PrimeFlex из props.class уже применятся через CSS
            let style = baseStyle;
            if (props.gap)
                style += `gap:${props.gap};`;
            if (props.align)
                style += `align-items:${props.align};`;
            if (props.justify)
                style += `justify-content:${props.justify};`;
            if (props.wrap)
                style += `flex-wrap:${props.wrap};`;
            if (props.style)
                style += props.style;
            el.style.cssText = style;
            // children will be appended by engine's dom handling for plugin? For layout, we need to handle children
            // The engine will append VNode children to this container via r.createElement's appendInto for plugin components,
            // but our tryMountPlugin currently handles children via r.createElement loop.
            // For layout, we just need a container that will hold children.
            container.appendChild(el);
            // store reference for update
            el.__props = props;
            return {
                update(newProps) {
                    let s = baseStyle;
                    if (newProps.gap)
                        s += `gap:${newProps.gap};`;
                    el.style.cssText = s + (newProps.style || "");
                },
                unmount() { el.remove(); }
            };
        }
    };
}
const Row = mkFlex("div", "display:flex;flex-direction:row;");
const Col = mkFlex("div", "display:flex;flex-direction:column;");
const Grid = {
    mount(container, props) {
        injectFlexCSS();
        const el = document.createElement("div");
        el.style.display = "grid";
        el.style.gap = props.gap || "8px";
        if (props.cols)
            el.style.gridTemplateColumns = `repeat(${props.cols}, 1fr)`;
        if (props.style)
            el.style.cssText += props.style;
        el.className = props.class || "";
        container.appendChild(el);
        return { update(p) { el.style.gap = p.gap || "8px"; }, unmount() { el.remove(); } };
    }
};
const Stack = mkFlex("div", "display:flex;flex-direction:column;");
const Container = {
    mount(container, props) {
        const el = document.createElement("div");
        el.className = "p-container " + (props.class || "");
        el.style.maxWidth = props.maxWidth || "1200px";
        el.style.margin = "0 auto";
        el.style.padding = props.padding || "16px";
        if (props.style)
            el.style.cssText += props.style;
        container.appendChild(el);
        return { update() { }, unmount() { el.remove(); } };
    }
};
const LayoutComponents = {
    Row, row: Row,
    Col, col: Col,
    Grid, grid: Grid,
    Stack, stack: Stack,
    Container, container: Container,
    Flex: Row, flex: Row,
};
function registerLayout() {
    window.UIEngineModules = window.UIEngineModules || {};
    Object.entries(LayoutComponents).forEach(([name, comp]) => {
        window.UIEngineModules[name] = comp;
    });
    if (window.UIEngine) {
        Object.keys(LayoutComponents).forEach(name => {
            window.UIEngine.registerComponent(name, LayoutComponents[name]);
        });
    }
}
if (typeof window !== "undefined") {
    const tryReg = () => {
        if (window.UIEngine)
            registerLayout();
        else
            setTimeout(tryReg, 100);
    };
    tryReg();
    injectFlexCSS();
}
export {};
//# sourceMappingURL=index.js.map