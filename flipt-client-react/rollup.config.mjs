import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import typescript from '@rollup/plugin-typescript';
import dts from 'rollup-plugin-dts';
import peerDepsExternal from 'rollup-plugin-peer-deps-external';

export default [
  {
    input: 'src/index.ts',
    output: [
      {
        dir: 'dist',
        entryFileNames: '[name].mjs',
        format: 'esm',
        sourcemap: true
      },
      {
        dir: 'dist',
        entryFileNames: '[name].cjs',
        format: 'cjs',
        sourcemap: true
      }
    ],
    plugins: [
      peerDepsExternal(),
      resolve(),
      commonjs(),
      typescript({ 
        tsconfig: './tsconfig.json',
        exclude: ['**/*.test.ts', '**/*.test.tsx']
      })
    ],
    external: ['react', 'react-dom', '@flipt-io/flipt-client-js']
  },
  {
    input: 'dist/types/index.d.ts',
    output: [{ dir: 'dist/types', format: 'esm' }],
    plugins: [dts()],
    external: ['react', 'react-dom', '@flipt-io/flipt-client-js']
  }
];
