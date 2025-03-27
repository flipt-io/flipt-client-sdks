import typescript from '@rollup/plugin-typescript';
import { wasm } from '@rollup/plugin-wasm';

export default {
  input: 'src/index.ts',
  output: [
    {
      file: 'dist/index.mjs',
      format: 'esm',
      sourcemap: true
    },
    {
      file: 'dist/index.cjs',
      format: 'cjs',
      sourcemap: true
    }
  ],
  plugins: [
    typescript(),
    wasm({
      targetEnv: 'auto-inline'
    })
  ]
};
