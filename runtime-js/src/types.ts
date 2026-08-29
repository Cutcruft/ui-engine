export interface ComponentHandle {
  mount(container: HTMLElement, props: Record<string, string>, onEvent: (action: string) => void): { update: (props: Record<string, string>) => void; unmount: () => void } | void | Promise<{ update: (props: Record<string, string>) => void; unmount: () => void } | void>;
  update?: (container: HTMLElement, props: Record<string, string>) => void;
  unmount?: (container: HTMLElement) => void;
}

export interface UIEngine {
  registerComponent(name: string, handle: ComponentHandle): void;
  getComponent(name: string): ComponentHandle | undefined;
  dispatch(action: string): void;
  getState(path: string): any;
  setState(path: string, value: any): void;
  registerAction(name: string, fn: () => void): void;
}

declare global {
  interface Window {
    UIEngine: UIEngine;
    UIEngineModules: Record<string, ComponentHandle>;
  }
}
