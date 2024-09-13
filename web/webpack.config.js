const path = require("path");
const { merge } = require("webpack-merge");
const webpack = require("webpack");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const MonacoWebpackPlugin = require("monaco-editor-webpack-plugin");
const NodePolyfillPlugin = require("node-polyfill-webpack-plugin");

// const { BundleAnalyzerPlugin } = require("webpack-bundle-analyzer");

function mapEnvironment(env) {
  if (env.enterprise) {
    return "enterprise";
  }
  return "dev";
}

// TODO: refactor this to use proper env varibles from webpack https://webpack.js.org/guides/environment-variables/
module.exports = function (env) {
  const distPath = path.join(__dirname, "dist");
  const srcPath = path.join(__dirname, "src");
  const appEnv = require(`./env/${mapEnvironment(env)}.js`);
  const replace = {};
  Object.entries(appEnv).forEach(
    ([key, value]) => (replace[`process.env.${key}`] = JSON.stringify(value))
  );
  if (env.enterprise) {
    process.env.NODE_ENV = "production";
  }

  const common = {
    output: {
      path: distPath,
      publicPath: "/",
      filename: "[name].[fullhash].js",
    },
    resolve: {
      extensions: [
        ".js",
        ".mjs",
        ".jsx",
        ".css",
        ".scss",
        ".png",
        ".jpg",
        ".svg",
        ".ico",
        ".tsx",
        ".ts",
      ],
      fallback: {
        fs: false,
        stream: require.resolve("stream-browserify"),
        crypto: require.resolve("crypto-browserify"),
        zlib: require.resolve("browserify-zlib"),
        constants: require.resolve("constants-browserify"),
        util: require.resolve("util/"),
        os: require.resolve("os-browserify/browser"),
        tty: require.resolve("tty-browserify"),
      },
      alias: {
        "@components": path.resolve(__dirname, "src/components"),
        "@features": path.resolve(__dirname, "src/features"),
        "@stores": path.resolve(__dirname, "src/stores"),
        "@types": path.resolve(__dirname, "src/types/index"),
        "@utils": path.resolve(__dirname, "src/utilities/index"),
        handlebars: "handlebars/dist/handlebars.js",
        "@src": path.resolve(__dirname, "src"),
      },
      mainFields: ["browser", "main"],
    },
    module: {
      rules: [
        {
          test: /\.mjs$/,
          include: /node_modules/,
          type: "javascript/auto",
        },
        {
          test: /\.css$/,
          use: [
            "style-loader",
            // { loader: MiniCssExtractPlugin.loader },
            "css-loader",
            "postcss-loader",
          ],
          sideEffects: true,
        },
        {
          test: /\.ttf$/,
          use: [{ loader: "file-loader" }],
          sideEffects: true,
        },
        {
          test: /\.scss$/,
          include: srcPath,
          use: [
            { loader: "style-loader" },
            // { loader: MiniCssExtractPlugin.loader },
            { loader: "css-loader", options: { importLoaders: 1 } },
            { loader: "postcss-loader" },
            { loader: "sass-loader" },
          ],
          sideEffects: true,
        },
        // {
        //   test:  /\.(sa|sc|c)ss$/,
        //   use: [
        //     //MiniCssExtractPlugin.loader,
        //     // { loader: "css-hot-loader" },
        //     { loader: "style-loader", options: { injectType: "styleTag" } },
        //     { loader: "css-loader" },
        //     { loader: "postcss-loader" },
        //     { loader: "sass-loader" },
        //   ],
        //   sideEffects: true,
        // },
        {
          test: /\.(png|jpg|svg|ico)$/,
          include: srcPath,
          type: "asset/resource",
        },
        {
          test: /\.woff(2)?(\?v=\d+\.\d+\.\d+)?$/,
          use: [
            {
              loader: "url-loader",
              options: {
                limit: 10000,
                mimetype: "application/font-woff",
                name: "./assets/[fullhash].[ext]",
              },
            },
          ],
        },
        {
          test: /\.m?js/,
          resolve: {
            fullySpecified: false,
          },
        },
      ],
    },
    plugins: [
      new NodePolyfillPlugin(),
      new webpack.ProvidePlugin({
        Buffer: ["buffer", "Buffer"],
        process: "process/browser",
      }),
      new webpack.DefinePlugin(replace),
      new HtmlWebpackPlugin({
        title: "Admin Console",
        inject: "body",
      }),
      new MonacoWebpackPlugin({
        languages: ["yaml", "json"],
        features: [
          "coreCommands",
          "folding",
          "bracketMatching",
          "clipboard",
          "find",
          "colorDetector",
          "codelens",
        ],
      }),
      new webpack.ContextReplacementPlugin(
        /graphql-language-service-interface[/\\]dist/,
        /\.js$/
      ),
      new MiniCssExtractPlugin({
        filename: "style.[fullhash].css",
        chunkFilename: "[id].css",
      }),
      // new BundleAnalyzerPlugin({
      //   generateStatsFile: true,
      //   analyzerHost: "0.0.0.0",
      //   analyzerPort: 30088
      // })
    ],
  };

  if (env.enterprise) {
    var dist = require("./webpack.config.dist");
    return merge(common, dist(appEnv));
  } else {
    var dev = require("./webpack.config.dev");
    return merge(common, dev);
  }
};
