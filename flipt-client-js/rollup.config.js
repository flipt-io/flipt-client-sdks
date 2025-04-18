import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';
import { copyFileSync, mkdirSync, existsSync, rmSync, readdirSync, statSync } from 'fs';
import glob from 'glob';
import path from 'path';

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

// Plugin to clean up duplicate WASM files
const cleanupWasmPlugin = (targetDir) => ({
  name: 'cleanup-wasm',
  writeBundle() {
    // Remove any duplicated WASM related files in the target directory
    const targetWasmFile = `dist/${targetDir}/flipt_engine_wasm_js*`;
    const targetWasmFiles = glob.sync(targetWasmFile);
    targetWasmFiles.forEach((file) => {
      if (existsSync(file)) {
        rmSync(file);
      }
    });
  }
});

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
    copyWasmPlugin('browser'),
    wasm({
      targetEnv: 'auto-inline'
    }),
    cleanupWasmPlugin('browser')
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
    copyWasmPlugin('node'),
    wasm({
      maxFileSize: 0,
      publicPath: "../",
      targetEnv: "node",
      fileName: "[name][extname]",
    }),
    cleanupWasmPlugin('node')
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
    }),
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
