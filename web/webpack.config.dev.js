var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");

module.exports = {
  mode: "development",
  entry: [
    "react-hot-loader/patch",
    "./src/index.jsx"
  ],

  plugins: [
    new webpack.NamedModulesPlugin(),
  ],

  output: {
    path: path.join(__dirname, "dist")
  },

  module: {
    rules: [
      {
        test: /\.[jt]sx?$/,
        include: srcPath,
        exclude: /node_modules/,
        enforce: "pre",
        loaders: ["babel-loader"],
      },
      {
        test: /\.[jt]sx?$/,
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
    compress: true,
    host: "0.0.0.0",
    hot: true,
    hotOnly: true,
    disableHostCheck: true,
    historyApiFallback: {
      verbose: true,
      disableDotRule: true
    },
  }
}
