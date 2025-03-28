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
      targetEnv: 'auto-inline'
    })
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
      targetEnv: 'auto-inline'
    })
  ],
  external: ['node-fetch']
};

export default [browserConfig, nodeConfig];
