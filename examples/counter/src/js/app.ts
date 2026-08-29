// Bootstrap TS: загружает wasm-ядро (плагины грузятся отдельно через index.html)

declare const Go: any;

async function loadConfigs(): Promise<any> {
  const resp = await fetch('/__config__');
  if (!resp.ok) throw new Error('config fetch failed');
  return resp.json();
}

async function main(): Promise<void> {
  const cfg = await loadConfigs();
  (window as any).__UI_CONFIG__ = cfg;

  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(
    fetch('/wasm/main.wasm'),
    go.importObject,
  );
  go.run(result.instance);
  (window as any).__go = go;

  // JS Bridge демо
  const tryRegister = () => {
    const UIEngine = (window as any).UIEngine;
    if (UIEngine) {
      UIEngine.registerAction("jsHello", () => {
        console.log("JS Bridge: jsHello вызван");
        UIEngine.dispatch("inc state.counter.count");
        UIEngine.setState("state.ui.jsBridgeFired", true);
      });
      console.log("JS Bridge: jsHello зарегистрирован");
    } else {
      setTimeout(tryRegister, 200);
    }
  };
  tryRegister();
}

main().catch((e) => {
  console.error('ui-engine bootstrap error', e);
});
