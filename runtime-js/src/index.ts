import type { UIEngine, ComponentHandle } from "./types";

// UIEngine TS bridge — строгая изоляция плагинов
declare global {
  interface Window {
    UIEngine: UIEngine;
    UIEngineModules: Record<string, ComponentHandle>;
  }
}

class UIEngineImpl implements UIEngine {
  private components = new Map<string, ComponentHandle>();
  private actions = new Map<string, () => void>();

  registerComponent(name: string, handle: ComponentHandle) {
    this.components.set(name, handle);
    this.components.set(name.toLowerCase(), handle);
    window.UIEngineModules = window.UIEngineModules || {};
    window.UIEngineModules[name] = handle;
    window.UIEngineModules[name.toLowerCase()] = handle;
  }

  getComponent(name: string): ComponentHandle | undefined {
    return this.components.get(name) || this.components.get(name.toLowerCase());
  }

  dispatch(action: string) {
    // делегируется в Go через window.__goDispatch (устанавливается wasm)
    const goDispatch = (window as any).__goDispatch as ((a: string) => void) | undefined;
    if (goDispatch) goDispatch(action);
    else console.warn("UIEngine.dispatch: Go not ready", action);
  }

  getState(path: string): any {
    const fn = (window as any).__goGetState as ((p: string) => any) | undefined;
    return fn ? fn(path) : null;
  }

  setState(path: string, value: any) {
    const fn = (window as any).__goSetState as ((p: string, v: any) => void) | undefined;
    if (fn) fn(path, value);
  }

  registerAction(name: string, fn: () => void) {
    this.actions.set(name, fn);
    this.actions.set(`action.${name}`, fn);
    // также регистрируем в Go
    const goReg = (window as any).__goRegisterAction as ((k: string, fn: () => void) => void) | undefined;
    if (goReg) goReg(name, fn);
  }
}

// Инициализация
if (typeof window !== "undefined") {
  window.UIEngineModules = window.UIEngineModules || {};
  const engine = new UIEngineImpl();
  // сохраняем существующие методы от wasm, если уже есть
  const prev = (window as any).UIEngine as Partial<UIEngine> | undefined;
  window.UIEngine = engine as any;
  if (prev) {
    Object.assign(window.UIEngine, prev);
    // восстанавливаем компоненты
    Object.entries(window.UIEngineModules).forEach(([k, v]) => engine.registerComponent(k, v));
  }
  console.log("UIEngine TS bridge initialized");
}

// types are available via import type, no export needed for plain script
