var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");

module.exports = {
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
    port: 30000,
    host: "0.0.0.0",
    hot: true,
    hotOnly: true,
    historyApiFallback: {
      verbose: true,
      disableDotRule: true
    },
  }
}
