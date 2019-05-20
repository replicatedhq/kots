var path = require("path");
var webpackMerge = require("webpack-merge");
var webpack = require("webpack");
var HtmlWebpackPlugin = require("html-webpack-plugin");
var HtmlWebpackTemplate = require("html-webpack-template");
var ScriptExtHtmlWebpackPlugin = require("script-ext-html-webpack-plugin");
const MonacoWebpackPlugin = require("monaco-editor-webpack-plugin");
const FaviconsWebpackPlugin = require("favicons-webpack-plugin");

module.exports = function (env) {
  var distPath = path.join(__dirname, "dist");
  var srcPath = path.join(__dirname, "src");
  var appEnv = require("./env/" + (env || "dev") + ".js");

  if (process.env["GITHUB_CLIENT_ID"]) {
    appEnv.GITHUB_CLIENT_ID = process.env["GITHUB_CLIENT_ID"];
  }
  if (process.env["GITHUB_INSTALL_URL"]) {
    appEnv.GITHUB_INSTALL_URL = process.env["GITHUB_INSTALL_URL"];
  }
  if (process.env["SHIP_CLUSTER_API_SERVER"]) {
    appEnv.INSTALL_ENDPOINT = `${process.env["SHIP_CLUSTER_API_SERVER"]}/api/install`;
    appEnv.GRAPHQL_ENDPOINT = `${process.env["SHIP_CLUSTER_API_SERVER"]}/graphql`;
    appEnv.REST_ENDPOINT = `${process.env["SHIP_CLUSTER_API_SERVER"]}/api`;
    appEnv.SHIPINIT_ENDPOINT = `${process.env["SHIP_CLUSTER_API_SERVER"]}/api/v1/init/`;
    appEnv.SHIPUPDATE_ENDPOINT = `${process.env["SHIP_CLUSTER_API_SERVER"]}/api/v1/update/`;
    appEnv.SHIPEDIT_ENDPOINT = `${process.env["SHIP_CLUSTER_API_SERVER"]}/api/v1/edit/`;
  }
  if (process.env["SHIP_CLUSTER_WEB_URI"]) {
    appEnv.GITHUB_REDIRECT_URI = `${process.env["SHIP_CLUSTER_WEB_URI"]}/auth/github/callback`;
  }

  var common = {
    output: {
      path: distPath,
      publicPath: "/",
      filename: "[name].[hash].js"
    },

    resolve: {
      extensions: [".js", ".mjs", ".jsx", ".css", ".scss", ".png", ".jpg", ".svg", ".ico"],
      alias: {
        "react": path.resolve("node_modules/react"),
        "react-dom": path.resolve("node_modules/react-dom"),
      }
    },

    devtool: "eval-source-map",

    node: {
      "fs": "empty"
    },

    module: {
      rules: [
        {
          test: /\.mjs$/,
          include: /node_modules/,
          type: 'javascript/auto'
        },
        {
          test: /\.css$/,
          use: [
            "style-loader",
            "css-loader",
            {
              loader: "postcss-loader",
              options: {
                config: {
                  path: require.resolve("./postcss.config.js"),
                }
              }
            }
          ]
        },
        {
          test: /\.scss$/,
          include: srcPath,
          use: [
            { loader: "style-loader" },
            { loader: "css-loader", options: { importLoaders: 1 } },
            { loader: "sass-loader" },
            { loader: "postcss-loader" }
          ]
        },
        {
          test: /\.(png|jpg|ico)$/,
          include: srcPath,
          use: ["file-loader"],
        },
        {
          test: /\.svg/,
          include: srcPath,
          use: ["svg-url-loader"],
        },
        {
          test: /\.woff(2)?(\?v=\d+\.\d+\.\d+)?$/,
          loader: "url-loader?limit=10000&mimetype=application/font-woff&name=./assets/[hash].[ext]",
        },
      ],
    },

    plugins: [
      new HtmlWebpackPlugin({
        template: HtmlWebpackTemplate,
        title: "Replicated Ship",
        appMountId: "app",
        externals: [
          {
            "react-dom": {
              root: "ReactDOM",
              commonjs2: "react-dom",
              commonjs: "react-dom",
              amd: "react-dom"
            }
          },
          {
            "react": {
              root: "React",
              commonjs2: "react",
              commonjs: "react",
              amd: "react"
            }
          }
        ],
        scripts: appEnv.WEBPACK_SCRIPTS,
        inject: false,
        window: {
          env: appEnv,
        },
      }),
      new ScriptExtHtmlWebpackPlugin({
        sync: "ship-cloud.js",
        defaultAttribute: "async"
      }),
      new FaviconsWebpackPlugin(srcPath + "/favicon-64.png"),
      new webpack.DefinePlugin({
        "process.env.NODE_ENV": JSON.stringify(appEnv.ENVIRONMENT),
      }),
      new MonacoWebpackPlugin({
        languages: [
          "yaml",
          "json",
        ],
        features: [
          "coreCommands",
          "folding",
          "bracketMatching",
          "clipboard",
          "find",
        ],
      }),
      new webpack.LoaderOptionsPlugin({
        options: {
          postcss: [
            require("autoprefixer")
          ]
        },
      }),
      new webpack.ContextReplacementPlugin(/graphql-language-service-interface[\/\\]dist/, /\.js$/)
    ],
  };

  if (env === "skaffold" || !env) {
    var dev = require("./webpack.config.dev");
    return webpackMerge(common, dev);
  } else {
    var dist = require("./webpack.config.dist");
    return webpackMerge(common, dist(env));
  }
};
