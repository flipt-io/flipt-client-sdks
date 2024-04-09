// HACK: define a fetch function that will load the wasm module if requested, otherwise use the default (node) fetch
// This is necessary because this is running in a node environment, and the default fetch implementation does not support loading relative paths
global.fetch = async (url, options) => {
  if (url instanceof URL && url.pathname.endsWith('.wasm')) {
    const fs = require('fs');
    const path = require('path');
    return fs.readFileSync(
      path.resolve(__dirname, 'dist/flipt_engine_wasm_bg.wasm')
    ).buffer;
  }
  const fetch = require('node-fetch');
  return fetch(url, options);
};
