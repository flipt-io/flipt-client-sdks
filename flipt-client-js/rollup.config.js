import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';
import fs from 'fs';
import path from 'path';

// Create a plugin specifically for moving the WASM file
const moveWasmFile = () => {
  return {
    name: 'move-wasm-file',

    writeBundle: {
      sequential: true, // Make sure this runs after all files are written
      order: 'post', // Run this after all other writeBundle hooks
      handler: () => {
        const sourceWasmPath = path.resolve(
          'src/wasm/flipt_engine_wasm_js_bg.wasm'
        );
        const targetWasmPath = path.resolve(
          'dist/flipt_engine_wasm_js_bg.wasm'
        );

        if (fs.existsSync(sourceWasmPath)) {
          console.log(
            `Moving WASM file from ${sourceWasmPath} to ${targetWasmPath}`
          );

          // Ensure dist directory exists
          if (!fs.existsSync('dist')) {
            fs.mkdirSync('dist', { recursive: true });
          }

          // Copy the file
          fs.copyFileSync(sourceWasmPath, targetWasmPath);

          // Remove the original to truly move it
          fs.unlinkSync(sourceWasmPath);
        }
      }
    }
  };
};

const browserConfig = {
  input: 'src/browser/index.ts',
  output: [
    {
      file: 'dist/browser/index.mjs',
      format: 'esm'
    },
    {
      file: 'dist/browser/index.cjs',
      format: 'cjs'
    }
  ],
  plugins: [
    wasm({
      targetEnv: 'auto-inline'
    }),
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null
    })
  ]
};

const nodeConfig = {
  input: 'src/node/index.ts',
  output: [
    {
      file: 'dist/node/index.mjs',
      format: 'esm'
    },
    {
      file: 'dist/node/index.cjs',
      format: 'cjs'
    }
  ],
  plugins: [
    wasm({
      targetEnv: 'auto-inline'
    }),
    typescript({
      noEmit: true,
      declaration: false,
      declarationDir: null
    }),
    moveWasmFile() // Add our custom plugin at the end
  ],
  external: ['node-fetch']
};

export default [browserConfig, nodeConfig];
