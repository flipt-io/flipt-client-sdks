import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';
import fs from 'fs';
import path from 'path';

// Create a plugin specifically for moving the WASM file
const moveWasmFile = () => {
  return {
    name: 'move-wasm-file',
    // wasm-bindgen outputs a import.meta.url when using the web target.
      // rollup will either perserve the the statement when outputting an esm,
      // which will cause webpack < 5 to choke or it will output a
      // "require('url')", for other output types, causing more choking. Since
      // we want a downstream developer to either not worry about providing wasm
      // at all, or forcing them to deal with bundling, we resolve the import to
      // an empty string. This will error at runtime.
      resolveImportMeta: () => `""`,
    writeBundle: {
      sequential: true, // Make sure this runs after all files are written
      order: 'post',    // Run this after all other writeBundle hooks
      handler: () => {
        const sourceWasmPath = path.resolve('dist/node/flipt_engine_wasm_js_bg.wasm');
        const targetWasmPath = path.resolve('dist/flipt_engine_wasm_js_bg.wasm');
        
        if (fs.existsSync(sourceWasmPath)) {
          console.log(`Moving WASM file from ${sourceWasmPath} to ${targetWasmPath}`);
          
          // Ensure dist directory exists
          if (!fs.existsSync('dist')) {
            fs.mkdirSync('dist', { recursive: true });
          }
          
          // Copy the file
          fs.copyFileSync(sourceWasmPath, targetWasmPath);
          
          // Remove the original to truly move it
          fs.unlinkSync(sourceWasmPath);
        } else {
          console.log('WASM file not found in dist/node');
        }
      }
    }
  };
};

const browserConfig = {
  input: 'src/browser/index.ts',
  output: [
    {
      dir: 'dist/browser',
      format: 'esm',
      entryFileNames: '[name].mjs',
    },
    {
      dir: 'dist/browser',
      format: 'cjs',
      entryFileNames: '[name].cjs',
    }
  ],
  plugins: [
    wasm({
      targetEnv: 'auto-inline',
    }),
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null,
      rootDir: 'src',
      outDir: 'dist/browser'
    }),
  ]
};

const nodeConfig = {
  input: 'src/node/index.ts',
  output: [
    {
      dir: 'dist/node',
      format: 'esm',
      entryFileNames: '[name].mjs',
    },
    {
      dir: 'dist/node',
      format: 'cjs',
      entryFileNames: '[name].cjs',
    }
  ],
  plugins: [
    wasm({
      targetEnv: 'node',
      maxFileSize: 0,
      publicPath: '../',
      fileName: '[name][extname]',
    }),
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null,
      rootDir: 'src',
      outDir: 'dist/node'
    }),
    moveWasmFile() // Add our custom plugin at the end
  ],
  external: ['node-fetch']
};

export default [browserConfig, nodeConfig];
