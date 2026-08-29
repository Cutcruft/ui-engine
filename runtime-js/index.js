class UIEngineImpl {
    constructor() {
        this.components = new Map();
        this.actions = new Map();
    }
    registerComponent(name, handle) {
        this.components.set(name, handle);
        this.components.set(name.toLowerCase(), handle);
        window.UIEngineModules = window.UIEngineModules || {};
        window.UIEngineModules[name] = handle;
        window.UIEngineModules[name.toLowerCase()] = handle;
    }
    getComponent(name) {
        return this.components.get(name) || this.components.get(name.toLowerCase());
    }
    dispatch(action) {
        // делегируется в Go через window.__goDispatch (устанавливается wasm)
        const goDispatch = window.__goDispatch;
        if (goDispatch)
            goDispatch(action);
        else
            console.warn("UIEngine.dispatch: Go not ready", action);
    }
    getState(path) {
        const fn = window.__goGetState;
        return fn ? fn(path) : null;
    }
    setState(path, value) {
        const fn = window.__goSetState;
        if (fn)
            fn(path, value);
    }
    registerAction(name, fn) {
        this.actions.set(name, fn);
        this.actions.set(`action.${name}`, fn);
        // также регистрируем в Go
        const goReg = window.__goRegisterAction;
        if (goReg)
            goReg(name, fn);
    }
}
// Инициализация
if (typeof window !== "undefined") {
    window.UIEngineModules = window.UIEngineModules || {};
    const engine = new UIEngineImpl();
    // сохраняем существующие методы от wasm, если уже есть
    const prev = window.UIEngine;
    window.UIEngine = engine;
    if (prev) {
        Object.assign(window.UIEngine, prev);
        // восстанавливаем компоненты
        Object.entries(window.UIEngineModules).forEach(([k, v]) => engine.registerComponent(k, v));
    }
    console.log("UIEngine TS bridge initialized");
}
export {};
// types are available via import type, no export needed for plain script
//# sourceMappingURL=index.js.map