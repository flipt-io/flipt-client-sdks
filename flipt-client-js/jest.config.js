/** @type {import('jest').Config} */
export default {
  preset: 'ts-jest/presets/default-esm',
  testEnvironment: 'node',
  extensionsToTreatAsEsm: ['.ts'],
  moduleNameMapper: {
    // Unit tests run against src/ with the generated WASM glue stubbed out.
    '^\\.\\./wasm/flipt_engine_wasm_js\\.js$': '<rootDir>/test-stubs/wasm-js.js',
    '^\\.\\./wasm/flipt_engine_wasm_js_bg\\.wasm$': '<rootDir>/test-stubs/wasm-bg.js',
    '^(\\.{1,2}/.*)\\.js$': '$1'
  },
  transform: {
    '^.+\\.tsx?$': [
      'ts-jest',
      {
        useESM: true
      }
    ]
  },
  setupFiles: ['<rootDir>/jest.polyfills.js'],
  moduleFileExtensions: ['ts', 'js', 'html']
};
