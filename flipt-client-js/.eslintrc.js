module.exports = {
  root: true,
  parser: '@typescript-eslint/parser',
  plugins: ['@typescript-eslint'],
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended'
  ],
  parserOptions: {
    project: './tsconfig.json',
    tsconfigRootDir: __dirname,
    ecmaVersion: 2020,
    sourceType: 'module'
  },
  env: {
    node: true,
    browser: true,
    es2020: true
  },
  rules: {
    '@typescript-eslint/no-explicit-any': 'off', // Since we need to handle WASM data
    '@typescript-eslint/explicit-module-boundary-types': 'off', // Due to WASM integration
    '@typescript-eslint/no-unused-vars': ['error', { 'argsIgnorePattern': '^_' }],
    '@typescript-eslint/no-unsafe-assignment': 'off', // Due to WASM integration
    '@typescript-eslint/no-unsafe-member-access': 'off', // Due to WASM integration
    '@typescript-eslint/no-unsafe-call': 'off', // Due to WASM integration
    '@typescript-eslint/restrict-template-expressions': 'off', // Allow template literals with any type
    '@typescript-eslint/no-inferrable-types': 'off', // Allow explicit type annotations
    '@typescript-eslint/ban-ts-comment': 'off', // Allow ts-ignore
    '@typescript-eslint/ban-types': ['error', {
      types: {
        '{}': false
      }
    }],
    '@typescript-eslint/no-empty-interface': 'off', // Allow empty interfaces for future extensibility
    'prefer-const': 'error',
    'no-console': ['error', { allow: ['warn', 'error'] }]
  },
  ignorePatterns: ['dist/', 'node_modules/', '*.js', '*.mjs']
}; 