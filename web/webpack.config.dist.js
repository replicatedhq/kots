var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");
var UglifyJsPlugin = require("uglifyjs-webpack-plugin");
const { BugsnagSourceMapUploaderPlugin } = require("webpack-bugsnag-plugins");

function getPlugins(appEnv) {
  const plugins = [
    new UglifyJsPlugin({
      uglifyOptions: {
        compress: { warnings: false },
        output: {
          comments: false,
        },
        minimize: false
      },
      sourceMap: true,
    }),
    new webpack.NamedModulesPlugin(),
  ];

  if (appEnv.BUGSNAG_API_KEY) {
    plugins.push(new BugsnagSourceMapUploaderPlugin({
      apiKey: appEnv.BUGSNAG_API_KEY,
      appVersion: "1.0.0",
      releaseStage: appEnv.ENVIRONMENT,
      overwrite: true
    }));
  }

  return plugins;
}

module.exports = (env) => {
  var appEnv = require("./env/" + (env || "dev") + ".js");

  return {
    entry: [
      "./src/index.jsx"
    ],

    module: {
      rules: [
        {
          test: /\.(js|jsx)$/,
          include: srcPath,
          exclude: /node_modules/,
          enforce: "pre",
          loaders: ["babel-loader"],
        },
      ],
    },

    plugins: getPlugins(appEnv),

    devtool: "source-map",

    stats: {
      colors: true,
      reasons: false
    }
  }
}
