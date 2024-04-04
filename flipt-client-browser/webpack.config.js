const path = require('path');

module.exports = {
  entry: {
    wasm: ['./src/bootstrap.ts']
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: 'ts-loader',
        exclude: /node_modules/
      }
    ]
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.js']
  },
  output: {
    path: path.resolve(__dirname, 'dist'),
    publicPath: ''
  },
  experiments: {
    asyncWebAssembly: true
  }
};
