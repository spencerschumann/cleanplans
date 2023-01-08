const CopyWebpackPlugin = require("copy-webpack-plugin");
const path = require('path');

module.exports = {
  entry: "./bootstrap.js",
  output: {
    path: path.resolve(__dirname, "dist"),
    filename: "bootstrap.js",
  },
  mode: "development",
  experiments: {
    asyncWebAssembly: true,
  },
  module: {
    rules: [
      {
        test: /\.js$/,
        enforce: "pre",
        use: ["source-map-loader"]
      }
    ]
  },
  plugins: [
    new CopyWebpackPlugin({
      patterns: [
        // TODO: this seems really nasty, there's got to be a better way.
        { from: 'index.html', to: 'index.html' },
        { from: 'favicon.ico', to: 'favicon.ico' },
        { from: 'wasm_exec.js', to: 'wasm_exec.js' },
        { from: 'go.wasm', to: 'go.wasm' },
      ]
    }),
  ],
};
