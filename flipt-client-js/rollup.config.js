import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';

const browserConfig = {
  input: 'src/browser/index.ts',
  output: [
    {
      file: 'dist/browser/index.mjs',
      format: 'esm',
      sourcemap: true
    },
    {
      file: 'dist/browser/index.cjs',
      format: 'cjs',
      sourcemap: true
    }
  ],
  plugins: [
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null
    }),
    wasm({
      targetEnv: 'auto-inline',
      maxFileSize: 0,
      publicPath: '',
      fileName: '[name][extname]',
      sync: ['./src/wasm/flipt_engine_wasm_js_bg.wasm']
    }),
  ]
};

const nodeConfig = {
  input: 'src/node/index.ts',
  output: [
    {
      file: 'dist/node/index.mjs',
      format: 'esm',
      sourcemap: true
    },
    {
      file: 'dist/node/index.cjs',
      format: 'cjs',
      sourcemap: true
    }
  ],
  plugins: [
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null
    }),
    wasm({
      targetEnv: 'node',
      maxFileSize: 0,
      publicPath: '',
      fileName: '[name][extname]',
      sync: ['./src/wasm/flipt_engine_wasm_js_bg.wasm']
    }),
  ],
  external: ['node-fetch']
};

export default [browserConfig, nodeConfig];
