const ReactRefreshWebpackPlugin = require("@pmmmwh/react-refresh-webpack-plugin");
const ESLintPlugin = require("eslint-webpack-plugin");
const webpack = require("webpack");
const path = require("path");

module.exports = {
  mode: "development",
  entry: ["./src/index.tsx"],
  cache: {
    type: "filesystem",
  },
  output: {
    path: path.join(__dirname, "dist"),
  },
  module: {
    rules: [
      {
        test: /\.[jt]sx?$/,
        exclude: /node_modules/,
        loader: "babel-loader",
        options: {
          plugins: [require.resolve("react-refresh/babel")],
        },
      },
    ],
  },
  plugins: [
    new webpack.HotModuleReplacementPlugin(),
    new ReactRefreshWebpackPlugin(),
    new ESLintPlugin(),
  ],
  optimization: {
    moduleIds: "named",
  },
  devtool: "eval",
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
      disableDotRule: true,
    },
  },
};
