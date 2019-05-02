var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");

module.exports = {
  entry: [
    "./src/index.jsx"
  ],

  plugins: [
    new webpack.NamedModulesPlugin(),
    new webpack.HotModuleReplacementPlugin(),
  ],

  output: {
    path: path.join(__dirname, 'dist')
  },

  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        include: srcPath,
        exclude: /node_modules/,
        enforce: "pre",
        loaders: ["babel-loader"],
      },
      {
        test: /\.(js|jsx)$/,
        include: srcPath,
        exclude: [
          /node_modules/,
        ],
        enforce: "pre",
        loaders: "eslint-loader",
        options: {
          fix: true
        }
      }
    ]
  },

  devtool: "eval-source-map",

  devServer: {
    port: 8000,
    hot: true,
    hotOnly: true,
    historyApiFallback: {
      verbose: true
    },
  }
}
