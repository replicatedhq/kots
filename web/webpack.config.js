const path = require("path");
const webpackMerge = require("webpack-merge");
const webpack = require("webpack");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const HtmlWebpackTemplate = require("html-webpack-template");
const CopyWebpackPlugin = require("copy-webpack-plugin");
const ScriptExtHtmlWebpackPlugin = require("script-ext-html-webpack-plugin");
const FaviconsWebpackPlugin = require("favicons-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");

/**
 * In the latest version of monaco-editor (0.26.x), monaco is using
 * a background web worker to make the UI pretty and fast, but
 * this internal code splitting broke a ton of stuff. To avoid this,
 * we can just not import the webpack plugin and use the peer
 * dependency inside of react-monaco-editor to just keep things working(tm)
 */
// const MonacoWebpackPlugin = require("monaco-editor-webpack-plugin");

// const { BundleAnalyzerPlugin } = require("webpack-bundle-analyzer");

module.exports = function (env) {
  const distPath = path.join(__dirname, "dist");
  const srcPath = path.join(__dirname, "src");
  const appEnv = require("./env/" + (env || "dev") + ".js");

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
        "/^monaco-editor/": "monaco-editor/esm/vs/editor/editor.api.js",
        "@src": path.resolve(__dirname, "src")
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
          type: "javascript/auto"
        },
        {
          test: /\.css$/,
          use: [
            "style-loader",
            {
              loader: MiniCssExtractPlugin.loader,
              options: {
                hmr: env === "skaffold",
              },
            },
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
            {
              loader: MiniCssExtractPlugin.loader,
              options: {
                hmr: env === "skaffold",
              },
            },
            { loader: "css-loader", options: { importLoaders: 1 } },
            { loader: "sass-loader" },
            {
              loader: "postcss-loader",
              options: {
                ident: "postcss",
                plugins: () => [
                  require("cssnano")()
                ]
              }
            }
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
          use: [
            {
              loader: "svg-url-loader",
              options: {
                stripdeclarations: true
              }
            }
          ],
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
      new FaviconsWebpackPlugin({
        logo: srcPath + "/favicon-64.png",
        icons: {
          android: false,
          appleIcon: false,
          appleStartup: false,
          coast: false,
          favicons: true,
          firefox: true,
          opengraph: false,
          twitter: false,
          yandex: false,
          windows: false
        }
      }),
      new webpack.DefinePlugin({
        "process.env.NODE_ENV": JSON.stringify(appEnv.ENVIRONMENT),
      }),
      new CopyWebpackPlugin([
        {
          from: "./src/services/prodPerfect.js",
          transform: function (content) {
            var contentS = content.toString("utf8");
            contentS = contentS.replace("@@PROD_PERFECT_WRITE_KEY", appEnv.PROD_PERFECT_WRITE_KEY);
            return contentS.toString(new Buffer(contentS));
          }
        }
      ]),
      new webpack.LoaderOptionsPlugin({
        options: {
          postcss: [
            require("autoprefixer")
          ]
        },
      }),
      new webpack.ContextReplacementPlugin(/graphql-language-service-interface[\/\\]dist/, /\.js$/),
      new MiniCssExtractPlugin({
        filename: "style.[hash].css",
        chunkFilename: "[id].css"
      })
      // new BundleAnalyzerPlugin({
      //   generateStatsFile: true,
      //   analyzerHost: "0.0.0.0",
      //   analyzerPort: 30088
      // })
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
