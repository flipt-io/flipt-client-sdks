import commonjs from '@rollup/plugin-commonjs';

export default {
  input: 'src/index.js',
  output: [
    {
      file: 'src/index.mjs',
      format: 'esm',
      sourcemap: true
    }
  ],
  plugins: [commonjs()]
};
