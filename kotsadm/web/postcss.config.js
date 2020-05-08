module.exports = {
  ident: "postcss",
  plugins: [
      require("autoprefixer"),
      require("cssnano")()
  ]
}