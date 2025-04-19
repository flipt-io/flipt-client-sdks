import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';
import fs from 'fs';
import path from 'path';

// Create a plugin specifically for creating dummy WASM files and copying the original WASM file to dist root
const copyWasmFiles = () => {
  return {
    name: 'copy-wasm-files',

    writeBundle: {
      sequential: true, // Make sure this runs after all files are written
      order: 'post', // Run this after all other writeBundle hooks
      handler: (options) => {
        // Extract the output directory from the output path
        const outputDir = path.dirname(options.file || '');

        // Ensure output directory exists
        if (!fs.existsSync(outputDir)) {
          fs.mkdirSync(outputDir, { recursive: true });
        }

        // Create the dummy WASM file in the same directory as the output
        const dummyWasmPath = path.join(
          outputDir,
          'flipt_engine_wasm_js_bg.wasm'
        );

        if (!fs.existsSync(dummyWasmPath)) {
          console.log(`Creating dummy WASM file at ${dummyWasmPath}`);
          // Create a minimal valid WASM file (8 bytes - magic number + version)
          const dummyWasmContent = Buffer.from([
            0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00
          ]);
          fs.writeFileSync(dummyWasmPath, dummyWasmContent);
        }

        // Also ensure we have the original WASM file copied to dist root
        const sourceWasmPath = path.resolve(
          'src/wasm/flipt_engine_wasm_js_bg.wasm'
        );
        const targetWasmPath = path.resolve(
          'dist/flipt_engine_wasm_js_bg.wasm'
        );

        if (fs.existsSync(sourceWasmPath) && !fs.existsSync(targetWasmPath)) {
          console.log(`Copying original WASM file to ${targetWasmPath}`);

          // Ensure dist directory exists
          if (!fs.existsSync('dist')) {
            fs.mkdirSync('dist', { recursive: true });
          }

          // Copy the file
          fs.copyFileSync(sourceWasmPath, targetWasmPath);
        }
      }
    }
  };
};

// Final cleanup plugin to remove everything from root dist except WASM files and folders
const finalCleanupPlugin = {
  name: 'final-cleanup',
  writeBundle() {
    const distDir = 'dist';
    const entries = readdirSync(distDir);
    
    entries.forEach(entry => {
      const entryPath = path.join(distDir, entry);
      const isDirectory = statSync(entryPath).isDirectory();
      
      // Keep directories
      if (isDirectory) {
        return;
      }
      
      // Keep .wasm files 
      if (entry.endsWith('.wasm')) {
        return;
      }
      
      // Remove everything else
      rmSync(entryPath);
    });
  }
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
    }),
    copyWasmFiles() // Add our dummy WASM creator
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
    copyWasmFiles() // Add our dummy WASM creator
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
    typescript(tsConfig),
    // Add the final cleanup as the last plugin in the last config
    finalCleanupPlugin
  ],
  external: ['node-fetch']
};

export default [
  browserConfig, 
  nodeConfig, 
  slimConfig
];
