const path = require("path");
const srcPath = path.join(__dirname, "src");
const TerserPlugin = require("terser-webpack-plugin");

function getPlugins(appEnv) {
  const plugins = [];

  return plugins;
}

module.exports = (appEnv) => {
  return {
    mode: "production",
    entry: [
      "./src/index.jsx"
    ],

    module: {
      rules: [
        {
          test: /\.[jt]sx?$/,
          include: srcPath,
          exclude: /node_modules/,
          enforce: "pre",
          use: ["babel-loader"],
        },
      ],
    },

    plugins: getPlugins(appEnv),

    devtool: "hidden-source-map",

    optimization: {
      moduleIds: "named",
      minimizer: [new TerserPlugin({
        terserOptions: {
          warnings: false,
          output: {
            comments: false,
          }
        },
        parallel: true
      })],
    },

    stats: {
      colors: true,
      reasons: false
    }
  }
}
