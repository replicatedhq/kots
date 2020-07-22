const path = require("path");
const { merge } = require("webpack-merge");
const webpack = require("webpack");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const HtmlWebpackTemplate = require("html-webpack-template");
const ScriptExtHtmlWebpackPlugin = require("script-ext-html-webpack-plugin");
const FaviconsWebpackPlugin = require("favicons-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const MonacoWebpackPlugin = require("monaco-editor-webpack-plugin");

// const { BundleAnalyzerPlugin } = require("webpack-bundle-analyzer");

module.exports = function (env) {
  const distPath = path.join(__dirname, "dist");
  const srcPath = path.join(__dirname, "src");
  const appEnv = require("./env/" + (env || "dev") + ".js");

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
          use: [
            {
              loader: "url-loader",
              options: {
                limit: 10000,
                mimetype: "application/font-woff",
                name: "./assets/[hash].[ext]"
              }
            }
          ]
        },
      ],
    },

    plugins: [
      new HtmlWebpackPlugin({
        template: HtmlWebpackTemplate,
        title: "Admin Console",
        appMountId: "app",
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
        "process.env.NODE_ENV": JSON.stringify(appEnv.ENVIRONMENT === "staging"
          ? "production"
          : appEnv.ENVIRONMENT
        )
      }),
      new MonacoWebpackPlugin({
        languages: [
          "yaml",
          "json"
        ],
        features: [
          "coreCommands",
          "folding",
          "bracketMatching",
          "clipboard",
          "find",
          "colorDetector",
          "codelens"
        ]
      }),
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
    return merge(common, dev);
  } else {
    var dist = require("./webpack.config.dist");
    return merge(common, dist(env));
  }
};
