"use strict";
// Bootstrap TS: загружает wasm-ядро (плагины грузятся отдельно через index.html)
async function loadConfigs() {
    const resp = await fetch('/__config__');
    if (!resp.ok)
        throw new Error('config fetch failed');
    return resp.json();
}
async function main() {
    const cfg = await loadConfigs();
    window.__UI_CONFIG__ = cfg;
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(fetch('/wasm/main.wasm'), go.importObject);
    go.run(result.instance);
    window.__go = go;
    // JS Bridge демо
    const tryRegister = () => {
        const UIEngine = window.UIEngine;
        if (UIEngine) {
            UIEngine.registerAction("jsHello", () => {
                console.log("JS Bridge: jsHello вызван");
                UIEngine.dispatch("inc state.counter.count");
                UIEngine.setState("state.ui.jsBridgeFired", true);
            });
            console.log("JS Bridge: jsHello зарегистрирован");
        }
        else {
            setTimeout(tryRegister, 200);
        }
    };
    tryRegister();
}
main().catch((e) => {
    console.error('ui-engine bootstrap error', e);
});
//# sourceMappingURL=app.js.map