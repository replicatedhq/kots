module.exports = {
  presets: [
    "@babel/preset-env",
    ["@babel/preset-react", { runtime: "automatic" }],
    "@babel/preset-typescript",
  ],
  plugins: [
    ["@babel/plugin-transform-class-properties", { loose: true }],
    ["@babel/plugin-transform-optional-chaining", { loose: true }],
  ],
};
