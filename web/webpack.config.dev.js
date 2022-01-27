const ReactRefreshWebpackPlugin = require("@pmmmwh/react-refresh-webpack-plugin");
const webpack = require("webpack");
const path = require("path");
const srcPath = path.join(__dirname, "src");

module.exports = {
  mode: "development",
  entry: [
    "./src/index.jsx"
  ],
  cache: {
    type: "filesystem"
  },
  output: {
    path: path.join(__dirname, "dist")
  },
  module: {
    rules: [
      {
        test: /\.[jt]sx?$/,
        exclude: /node_modules/,
        loader: "babel-loader"
      },
      {
        test: /\.[jt]sx?$/,
        include: srcPath,
        exclude: [
          /node_modules/,
        ],
        enforce: "pre",
        loader: "eslint-loader",
        options: {
          fix: true
        }
      }
    ]
  },
  plugins: [
    new webpack.HotModuleReplacementPlugin(),
    new ReactRefreshWebpackPlugin(),
  ],
  optimization: {
    moduleIds: "named"
  },
  devtool: "eval-source-map",
  devServer: {
    compress: true,
    hot: true,
    host: "0.0.0.0",
    allowedHosts: ["all"],
    client: {
      webSocketURL: "auto://0.0.0.0/ws",
    },
    historyApiFallback: {
      verbose: true,
      disableDotRule: true
    },
  },
}
