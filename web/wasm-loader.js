(function () {
  "use strict";

  async function loadMinyoGoWasm(options) {
    const wasmURL = (options && options.wasmURL) || "./minyo-go.wasm";
    if (typeof Go !== "function") {
      throw new Error("Go WASM runtime is missing. Load wasm_exec.js first.");
    }

    const go = new Go();
    let result;
    try {
      result = await WebAssembly.instantiateStreaming(
        fetch(wasmURL),
        go.importObject
      );
    } catch (_) {
      const response = await fetch(wasmURL);
      if (!response.ok) {
        throw new Error("WASM download failed: HTTP " + response.status);
      }
      result = await WebAssembly.instantiate(
        await response.arrayBuffer(),
        go.importObject
      );
    }

    // The Go program installs window.MinyoGo before entering its event loop.
    go.run(result.instance);
    if (!window.MinyoGo) {
      throw new Error("Go WASM started but MinyoGo API was not registered.");
    }
    return window.MinyoGo;
  }

  window.loadMinyoGoWasm = loadMinyoGoWasm;
})();
