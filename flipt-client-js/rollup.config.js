import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';
import { copyFileSync, mkdirSync, existsSync, rmSync } from 'fs';
import glob from 'glob';

// Custom plugin to copy WASM files to output directories
const copyWasmPlugin = (targetDir) => ({
  name: 'copy-wasm',
  writeBundle() {
    const wasmFiles = glob.sync('dist/flipt_engine_wasm_js_bg.wasm');
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

// Plugin to clean up duplicate WASM files
const cleanupWasmPlugin = () => ({
  name: 'cleanup-wasm',
  writeBundle() {
    // Remove the WASM file from the root dist folder since it's already copied to common
    const rootWasmFile = 'dist/flipt_engine_wasm_js_bg.wasm';
    if (existsSync(rootWasmFile)) {
      rmSync(rootWasmFile);
    }

    // Remove any duplicated WASM files in the node directory
    const nodeWasmFile = 'dist/node/flipt_engine_wasm_js_bg.wasm';
    if (existsSync(nodeWasmFile)) {
      rmSync(nodeWasmFile);
    }
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
      maxFileSize: 0,
      publicPath: '../common/',
      targetEnv: 'node',
      fileName: '[name][extname]'
    }),
    copyWasmPlugin('common'),
    cleanupWasmPlugin()
  ],
  external: ['node-fetch']
};

// Slim configuration that doesn't bundle the WASM file
const slimConfig = {
  input: 'src/slim/index.ts',
  output: [
    {
      file: 'dist/slim/index.mjs',
      format: 'esm',
      sourcemap: true
    },
    {
      file: 'dist/slim/index.cjs',
      format: 'cjs',
      sourcemap: true
    }
  ],
  plugins: [
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null
    })
  ],
  external: ['node-fetch']
};

export default [browserConfig, nodeConfig, slimConfig];
