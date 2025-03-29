import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';
import { copyFileSync, mkdirSync, unlinkSync } from 'fs';
import { dirname } from 'path';
import glob from 'glob';

// Custom plugin to copy WASM files to output directories
const copyWasmPlugin = (targetDir) => ({
  name: 'copy-wasm',
  writeBundle() {
    const wasmFiles = glob.sync('dist/flipt_engine_wasm_js*');
    const targetPath = `dist/${targetDir}`;

    // Ensure target directory exists
    mkdirSync(targetPath, { recursive: true });

    // Copy all WASM-related files to target directory
    wasmFiles.forEach((file) => {
      const filename = file.split('/').pop();
      copyFileSync(file, `${targetPath}/${filename}`);
    });
  }
});

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
    }),
    copyWasmPlugin('browser')
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
    }),
    copyWasmPlugin('node')
  ],
  external: ['node-fetch']
};

export default [browserConfig, nodeConfig];
