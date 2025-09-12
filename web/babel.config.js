module.exports = {
  presets: [
    ["@babel/preset-env", {
      modules: process.env.NODE_ENV === 'test' ? 'commonjs' : false,
      targets: {
        browsers: ["> 1%", "last 4 versions", "not IE 11", "not dead"]
      }
    }],
    ["@babel/preset-react", { runtime: "automatic" }],
    "@babel/preset-typescript",
  ],
  plugins: [
    ["@babel/plugin-transform-class-properties", { loose: true }],
    ["@babel/plugin-transform-optional-chaining", { loose: true }],
  ],
};
